use crate::api::{Err, Sub};
use crate::cfg::{Cfg, Host};
use crate::out::{CYN, DIM, GRN, GRY, YLW, bx, ce, die, link, scol, stat, warn};
use crate::{fail, hct, need, spin, t};
use std::{
	path::Path,
	time::{Duration, Instant},
};

fn ext(s: &str) -> &str {
	let b = s.rsplit(['/', '\\']).next().unwrap_or(s); // ext = from the last dot of the last path elem
	b.rfind('.').map_or("", |i| &b[i..])
}

fn base(s: &str) -> &str {
	s.rsplit(['/', '\\']).next().unwrap_or(s)
}

pub fn guess(arg: &str) -> Result<(String, String), String> {
	let e = ext(arg); // -> prob, file
	if !e.is_empty() {
		return Ok((base(arg).strip_suffix(e).unwrap_or(base(arg)).into(), arg.into()));
	}
	if Path::new(&format!("{arg}.cpp")).exists() {
		return Ok((arg.into(), format!("{arg}.cpp")));
	}
	let (dir, b) = match arg.rfind(['/', '\\']) {
		Some(i) => (&arg[..=i], &arg[i + 1..]),
		None => ("", arg),
	};
	let mut m: Vec<String> = std::fs::read_dir(if dir.is_empty() { "." } else { dir })
		.map(|rd| rd.flatten().filter_map(|e| e.file_name().to_str().map(String::from)).filter(|n| n.starts_with(&format!("{b}."))).map(|n| format!("{dir}{n}")).collect())
		.unwrap_or_default();
	m.sort();
	match m.len() {
		1 => Ok((base(arg).into(), m.remove(0))),
		0 => Err(t!("no_file", arg)),
		_ => Err(t!("multi_file", format!("{arg}.*"))),
	}
}

fn resolve(arg: &str) -> (String, String) {
	guess(arg).unwrap_or_else(|e| die(&e))
}

pub const NO_API: &str = "!"; // Sub.status when the host cant list subs at all
pub const NO_SESS: &str = "!s"; // same but logged in and the session died

pub fn dur(d: Duration) -> String {
	let s = d.as_secs_f64().round() as u64; // 12s, 1m5s
	if s < 60 { format!("{s}s") } else { format!("{}m{}s", s / 60, s % 60) }
}

pub fn judge(h: &Host, ct: &str, p: &str, id: &str, tick: impl Fn(Duration)) -> (Sub, bool) {
	let (t0, mut dl, mut web) = (Instant::now(), Duration::from_millis(1500), false); // poll till id isnt pending, false = gave up
	while t0.elapsed() < Duration::from_secs(300) {
		std::thread::sleep(dl);
		dl = (dl + Duration::from_millis(500)).min(Duration::from_secs(4));
		tick(t0.elapsed());
		if web {
			let r = h.sub_web(ct, id); // old oioioi, read the my submissions page w/ the login cookie
			match r {
				Err(Err::Sess) => return (Sub { status: NO_SESS.into(), ..Default::default() }, true),
				Ok(s) if s.status != "?" => return (s, true),
				_ => continue,
			}
		}
		match h.subs(ct, p) {
			Err(e) if e.code() == 404 || matches!(e, Err::Html) => {
				if h.session.is_empty() {
					return (Sub { status: NO_API.into(), ..Default::default() }, true); // waiting wont help, needs sio config hosts login
				}
				(web, dl) = (true, Duration::ZERO); // old oioioi (camp) has no problem_submission_list
			}
			Err(_) => continue, // flaky net, keep trying
			Ok((ss, _)) => {
				if let Some(s) = ss.into_iter().find(|s| s.id.to_string() == id && s.status != "?") {
					return (s, true); // "" = hidden by the contest
				}
			}
		}
	}
	(Sub { status: "?".into(), ..Default::default() }, false)
}

pub fn verdict(s: &Sub) -> (String, String) {
	if s.status == NO_API || s.status == NO_SESS {
		return (t!("no_judge"), YLW.into()); // -> "✓ OK 100", color
	}
	if s.status.is_empty() {
		return (t!("hidden"), DIM.into());
	}
	let mut st = stat(&s.status);
	if let Some(n) = s.score {
		st.s += &format!(" {n}");
		st.c = scol(n).into();
	}
	(st.s, st.c)
}

pub fn run(c: &Cfg, args: &[String]) {
	let nw = args.iter().any(|a| a == "-n" || a == "--no-wait");
	let args: Vec<String> = args.iter().filter(|a| *a != "-n" && *a != "--no-wait").cloned().collect();
	need(&args, 2, "submit [-n] <[host/]contest> <prob|file> [file]");
	let (hn, h, ct) = hct(c, &args[0]);
	let (prob, file) = if args.len() > 2 { (args[1].clone(), args[2].clone()) } else { resolve(&args[1]) };
	let id = spin::wait(t!("w_up", file, format!("{hn}/{ct}/{prob}")), || h.submit(&ct, &prob, &file)).unwrap_or_else(|e| fail(&hn, &e));
	let u = format!("{}/c/{ct}/s/{id}/", h.url);
	let mut rows = vec![[t!("k_file"), file.clone()], [t!("k_prob"), format!("{hn}/{ct}/{prob}")], [t!("k_id"), id.clone()], [t!("k_url"), u.clone()]];
	if nw {
		bx(&format!("✓ {}", t!("sub_ok")), GRN, &rows);
		return;
	}
	let (s, done) = spin::wait(t!("w_judge", prob, "0s"), || judge(&h, &ct, &prob, &id, |d| spin::set(t!("w_judge", prob, dur(d)))));
	let (v, vc) = verdict(&s);
	rows.insert(3, [t!("k_stat"), v.clone()]);
	bx(&v, &vc, &rows);
	let web = || eprintln!("  {}", ce(GRY, &link(*crate::out::COL_ERR, &u, &t!("open_web"))));
	if s.status == NO_API {
		warn(&format!("{} {}", t!("no_judge_why", ce(CYN, &hn)), t!("need_login", ce(CYN, &format!("sio config hosts login {hn}")))));
		web();
	}
	if s.status == NO_SESS {
		warn(&t!("sess_exp", ce(CYN, &hn), ce(CYN, &format!("sio config hosts login {hn}"))));
		web();
	}
	if !done {
		warn(&t!("judge_slow", ce(CYN, &format!("sio subs {hn}/{ct} {prob}"))));
	}
}
