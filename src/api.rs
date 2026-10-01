use crate::cfg::{Host, nstr};
use chrono::{DateTime, FixedOffset};
use regex::Regex;
use serde::{Deserialize, Deserializer, de::DeserializeOwned};
use std::{fmt, sync::LazyLock, time::Duration};

#[derive(Debug, Clone)]
pub enum Err {
	Http(u16, String), // "404: Not Found"
	Html,              // block pages etc come back as 200 html
	Login,             // wrong user/pass
	Tfa,               // two_factor wants a token step
	Sess,              // not logged in or expired
	Net(String),
}

impl fmt::Display for Err {
	fn fmt(&self, f: &mut fmt::Formatter) -> fmt::Result {
		match self {
			Err::Http(c, s) => write!(f, "{c}: {s}"),
			Err::Html => f.write_str("not the api"),
			Err::Login => f.write_str("bad login"),
			Err::Tfa => f.write_str("needs 2fa"),
			Err::Sess => f.write_str("no session"),
			Err::Net(s) => f.write_str(s),
		}
	}
}

impl Err {
	pub fn code(&self) -> u16 {
		if let Err::Http(c, _) = self { *c } else { 0 } // 0 if not http
	}
}

pub type Res<T> = Result<T, Err>;

pub fn oserr(e: &std::io::Error) -> String {
	let s = e.to_string(); // "no such file or directory", no (os error 2)
	let s = s.split(" (os error").next().unwrap_or(&s);
	let mut c = s.chars();
	c.next().map_or(String::new(), |f| f.to_lowercase().chain(c).collect())
}

pub static AG: LazyLock<ureq::Agent> = LazyLock::new(|| ureq::AgentBuilder::new().timeout(Duration::from_secs(30)).build());

pub fn resp(r: Result<ureq::Response, ureq::Error>) -> Res<ureq::Response> {
	match r {
		Ok(r) | Err(ureq::Error::Status(_, r)) => Ok(r), // 4xx/5xx are still a response
		Err(e) => Err(Err::Net(e.to_string())),
	}
}

pub fn body(r: ureq::Response) -> String {
	r.into_string().unwrap_or_default()
}

#[derive(Deserialize, Clone, Default)]
pub struct Res0 {
	#[serde(default)]
	pub score: serde_json::Value, // int, null or raw "int:000100"
	#[serde(default, deserialize_with = "nstr")]
	pub status: String,
}

#[derive(Deserialize, Clone, Default)]
pub struct Prob {
	#[serde(rename = "short_name", default, deserialize_with = "nstr")]
	pub short: String,
	#[serde(rename = "full_name", default, deserialize_with = "nstr")]
	pub name: String,
	#[serde(rename = "submissions_left", default)]
	pub left: Option<i64>,
	#[serde(rename = "user_result", default)]
	pub res: Option<Res0>,
}

#[derive(Deserialize, Clone, Default)]
pub struct Sub {
	#[serde(default)]
	pub id: i64,
	#[serde(default, deserialize_with = "ndate")]
	pub date: Option<DateTime<FixedOffset>>,
	#[serde(default)]
	pub score: Option<i64>,
	#[serde(default, deserialize_with = "nstr")]
	pub status: String,
	#[serde(skip)]
	pub prob: String,
}

fn ndate<'de, D: Deserializer<'de>>(d: D) -> Result<Option<DateTime<FixedOffset>>, D::Error> {
	Ok(Option::<String>::deserialize(d)?.and_then(|s| DateTime::parse_from_rfc3339(&s).ok()))
}

#[derive(Deserialize, Clone, Default)]
pub struct Contest {
	#[serde(default, deserialize_with = "sid")]
	pub id: String,
	#[serde(default, deserialize_with = "nstr")]
	pub name: String,
}

fn sid<'de, D: Deserializer<'de>>(d: D) -> Result<String, D::Error> {
	Ok(match serde_json::Value::deserialize(d)? {
		serde_json::Value::String(s) => s, // id can be a str or a num
		serde_json::Value::Null => String::new(),
		v => v.to_string(),
	})
}

static PROB_RE: LazyLock<Regex> = LazyLock::new(|| Regex::new(r#"href="/c/[^/"]+/p/([a-z0-9_-]+)/"[^>]*>([^<]+)</a>"#).unwrap());

impl Host {
	fn req(&self, r: ureq::Request) -> ureq::Request {
		let r = r.set("Accept", "application/json");
		if self.token.is_empty() { r } else { r.set("Authorization", &format!("Token {}", self.token)) }
	}

	fn done(&self, r: Result<ureq::Response, ureq::Error>) -> Res<String> {
		let r = resp(r)?;
		let (c, js, st) = (r.status(), r.header("content-type").unwrap_or("").contains("json"), r.status_text().to_string());
		let b = body(r);
		if c / 100 != 2 {
			if let Ok(v) = serde_json::from_str::<serde_json::Value>(&b)
				&& let Some(d) = v.get("detail").or(v.get("Detail")).and_then(|d| d.as_str()).filter(|d| !d.is_empty())
			{
				return Err(Err::Http(c, d.into()));
			}
			if !js {
				return Err(Err::Http(c, st)); // html error page, dont dump it
			}
			return Err(Err::Http(c, b.trim().into()));
		}
		if !js {
			return Err(Err::Html);
		}
		Ok(b)
	}

	fn get<T: DeserializeOwned>(&self, p: &str) -> Res<T> {
		let b = self.done(self.req(AG.get(&format!("{}{p}", self.url))).call())?;
		serde_json::from_str(&b).map_err(|e| Err::Net(e.to_string()))
	}

	pub fn contests(&self) -> Res<Vec<Contest>> {
		self.get("/api/contest_list")
	}

	pub fn probs(&self, ct: &str) -> Res<Vec<Prob>> {
		self.get(&format!("/api/c/{ct}/problem_list/"))
	}

	pub fn probs_web(&self, ct: &str) -> Res<Vec<Prob>> {
		let r = resp(AG.get(&format!("{}/c/{ct}/p/", self.url)).call())?; // scrape /c/ct/p/ as anon, no scores
		let c = r.status();
		let b = body(r);
		let mut ps: Vec<Prob> = vec![];
		for m in PROB_RE.captures_iter(&b) {
			if !ps.iter().any(|p| p.short == m[1]) {
				ps.push(Prob { short: m[1].into(), name: html_escape::decode_html_entities(&m[2]).trim().into(), ..Default::default() });
			}
		}
		if ps.is_empty() {
			return Err(Err::Http(c, format!("no problems on {}/c/{ct}/p/", self.url)));
		}
		Ok(ps)
	}

	pub fn subs(&self, ct: &str, prob: &str) -> Res<(Vec<Sub>, bool)> {
		#[derive(Deserialize)]
		struct R {
			#[serde(rename = "submissions", default)]
			subs: Vec<Sub>,
			#[serde(rename = "is_truncated_to_20", default)]
			trunc: bool,
		}
		let mut r: R = self.get(&format!("/api/c/{ct}/problem_submission_list/{prob}/"))?; // last 20 only, bool = truncated
		for s in &mut r.subs {
			s.prob = prob.into();
		}
		Ok((r.subs, r.trunc))
	}

	pub fn ping(&self) -> Res<String> {
		let b = self.done(self.req(AG.get(&format!("{}/api/auth_ping", self.url))).call())?;
		let b = b.trim_matches(|c| c == '"' || c == '\n');
		Ok(b.strip_prefix("pong ").unwrap_or(b).into())
	}

	pub fn submit(&self, ct: &str, prob: &str, file: &str) -> Res<String> {
		let d = std::fs::read(file).map_err(|e| Err::Net(format!("open {file}: {}", oserr(&e))))?;
		let (bd, fname) = (format!("sio{:016x}", fastrand::u64(..)), std::path::Path::new(file).file_name().and_then(|n| n.to_str()).unwrap_or(file).replace(['\\', '"'], "_"));
		let mut b = format!("--{bd}\r\nContent-Disposition: form-data; name=\"file\"; filename=\"{fname}\"\r\nContent-Type: application/octet-stream\r\n\r\n").into_bytes();
		b.extend(d);
		b.extend(format!("\r\n--{bd}--\r\n").as_bytes());
		let r = self.req(AG.post(&format!("{}/api/c/{ct}/submit/{prob}", self.url))).set("Content-Type", &format!("multipart/form-data; boundary={bd}")).send_bytes(&b);
		Ok(self.done(r)?.trim().into())
	}
}
