use crate::api::{Err, Res, Sub, body, resp};
use crate::cfg::Host;
use regex::Regex;
use std::{sync::LazyLock, time::Duration};

static NF: LazyLock<ureq::Agent> = LazyLock::new(|| ureq::AgentBuilder::new().timeout(Duration::from_secs(30)).redirects(0).build()); // 3xx back as is
static CSRF: LazyLock<Regex> = LazyLock::new(|| Regex::new(r#"name="csrfmiddlewaretoken" value="([^"]+)""#).unwrap());
static OK: LazyLock<Regex> = LazyLock::new(|| Regex::new(r"^((?:INI_)?OK)\d+$").unwrap());
static TFA: LazyLock<Regex> = LazyLock::new(|| Regex::new(r#"current_step"[^>]*value="token"|value="token"[^>]*current_step"#).unwrap());
static CLS: LazyLock<Regex> = LazyLock::new(|| Regex::new(r"submission--(\S+)").unwrap());

fn cookie(r: &ureq::Response, n: &str) -> String {
	for c in r.all("set-cookie") {
		if let Some((k, v)) = c.split(';').next().unwrap_or("").split_once('=')
			&& k.trim() == n
		{
			return v.trim().into();
		}
	}
	String::new()
}

impl Host {
	pub fn login(&mut self, user: &str, pass: &str) -> Res<()> {
		let r = resp(NF.get(&format!("{}/login/", self.url)).call())?; // web login like the site, keeps only the sessionid
		let (c, ck) = (r.status(), cookie(&r, "csrftoken"));
		let b = body(r);
		let Some(m) = CSRF.captures(&b).filter(|_| !ck.is_empty()) else {
			return Err(Err::Http(c, format!("no login form on {}/login/", self.url)));
		};
		let f = [("csrfmiddlewaretoken", &m[1]), ("login_view-current_step", "auth"), ("auth-username", user), ("auth-password", pass)];
		let r = resp(NF.post(&format!("{}/login/", self.url)).set("Referer", &format!("{}/login/", self.url)).set("Cookie", &format!("csrftoken={ck}")).send_form(&f))?; // django csrf wants the referer on https
		let (c, s) = (r.status(), cookie(&r, "sessionid"));
		let b = body(r);
		if !s.is_empty() && c / 100 == 3 {
			self.session = s;
			return Ok(());
		}
		Err(if TFA.is_match(&b) { Err::Tfa } else { Err::Login })
	}

	pub fn sub_web(&self, ct: &str, id: &str) -> Res<Sub> {
		if self.session.is_empty() {
			return Err(Err::Sess); // read one row off /c/ct/submissions/, status "?" if not there yet
		}
		let r = resp(NF.get(&format!("{}/c/{ct}/submissions/", self.url)).set("Cookie", &format!("sessionid={}", self.session)).call())?;
		if r.status() / 100 == 3 {
			return Err(Err::Sess); // bounced to /login/, session expired
		}
		let (b, id) = (body(r), regex::escape(id));
		let mut s = Sub { status: "?".into(), ..Default::default() };
		let Some(st) = Regex::new(&format!(r#"id="submission{id}-status"\s*class="([^"]*)""#)).unwrap().captures(&b) else {
			return Ok(s);
		};
		s.status = String::new(); // row there but no submission--X = hidden
		if let Some(m) = CLS.captures(&st[1]) {
			s.status = OK.replace(&html_escape::decode_html_entities(&m[1]), "$1").into(); // display_type, ? while judging, OK100 -> OK (num = css score bucket)
		}
		if let Some(m) = Regex::new(&format!(r#"id="submission{id}-score"[^>]*>\s*([^<]*?)\s*<"#)).unwrap().captures(&b) {
			s.score = m[1].split_whitespace().next().and_then(|x| x.parse().ok());
		}
		Ok(s)
	}
}
