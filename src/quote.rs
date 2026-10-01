use serde::Deserialize;
use std::{
	fs,
	path::PathBuf,
	sync::LazyLock,
	time::{Duration, SystemTime},
};

const URL: &str = "https://github.com/mudroljub/programming-quotes-api/raw/refs/heads/master/data/quotes.json";
const WEEK: Duration = Duration::from_secs(7 * 24 * 3600);
static AG: LazyLock<ureq::Agent> = LazyLock::new(|| ureq::AgentBuilder::new().timeout(Duration::from_secs(3)).build()); // short, runs after every cmd

#[derive(Deserialize, Clone)]
pub struct Quote {
	#[serde(default)]
	pub text: String,
	#[serde(default)]
	pub author: String,
}

fn path() -> PathBuf {
	dirs::cache_dir().unwrap_or_default().join("sio").join("quotes.json")
}

fn age(p: &PathBuf) -> Option<Duration> {
	fs::metadata(p).ok()?.modified().ok()?.elapsed().ok()
}

pub fn stale() -> bool {
	age(&path()).is_none_or(|a| a > WEEK) // true = rand will hit the net
}

fn touch(p: &PathBuf, t: SystemTime) {
	if let Ok(f) = fs::File::options().write(true).open(p) {
		let _ = f.set_modified(t);
	}
}

pub fn rand() -> Option<Quote> {
	let p = path(); // cached a week in the user cache dir
	let a = age(&p);
	if a.is_none_or(|a| a > WEEK) {
		let _ = crate::cfg::mkdir(p.parent().unwrap());
		match fetch() {
			Some(b) => {
				let _ = crate::cfg::write(&p, &b);
			}
			None if a.is_none() => {
				let _ = crate::cfg::write(&p, b"[]"); // offline w/ no cache, retry in a day not every run
				touch(&p, SystemTime::now() - Duration::from_secs(6 * 24 * 3600));
			}
			None => touch(&p, SystemTime::now()), // keep the stale one a week more
		}
	}
	let qs: Vec<Quote> = serde_json::from_slice(&fs::read(&p).ok()?).ok()?;
	if qs.is_empty() { None } else { Some(qs[fastrand::usize(..qs.len())].clone()) }
}

fn fetch() -> Option<Vec<u8>> {
	let r = AG.get(URL).call().ok().filter(|r| r.status() == 200)?;
	let mut b = vec![];
	std::io::Read::read_to_end(&mut r.into_reader(), &mut b).ok()?;
	serde_json::from_slice::<Vec<Quote>>(&b).ok()?;
	Some(b)
}
