use crate::api::Err;
use crate::cfg::{Cfg, Host, cfg_path};
use crate::out::{COL_OUT, CYN, DIM, GRN, ce, cell, co, die, ok, tbl, warn};
use crate::{CFG_USAGE, ask_token, fail, hname, lang, line, need, spin, t};
use std::io::Write;

fn onoff(b: bool) -> String {
	if b { t!("on") } else { t!("off") }
}

fn save(c: &Cfg) {
	if let Err(e) = c.save() {
		die(&e.to_string());
	}
}

fn mark(on: bool) -> String {
	match (*COL_OUT, on) {
		(false, true) => "*".into(), // plain markers when piped, completions cut on bytes
		(false, false) => " ".into(),
		(true, true) => co(GRN, "●"),
		(true, false) => co(DIM, "○"),
	}
}

fn lang_env() {
	if let Ok(v) = std::env::var("SIO_LANG")
		&& !v.is_empty()
	{
		warn(&t!("lang_env", ce(CYN, &format!("SIO_LANG={v}"))));
	}
}

pub fn run(c: &mut Cfg, args: &[String]) {
	if args.is_empty() {
		let lc = lang::code(&c.lang);
		let src = if std::env::var("SIO_LANG").is_ok_and(|v| !v.is_empty()) {
			t!("src_env")
		} else if !c.lang.is_empty() {
			t!("src_saved")
		} else {
			t!("src_auto")
		};
		let mn = if c.main.is_empty() { "-".into() } else { c.main.clone() };
		let rows = vec![
			vec![cell(t!("k_cfg"), DIM), cell(cfg_path().display().to_string(), "")],
			vec![cell(t!("k_hosts"), DIM), cell(t!("hosts_sum", c.hosts.len(), mn), "")],
			vec![cell(t!("k_lang"), DIM), cell(format!("{lc} ({}), {src}", lang::name(&lc)), "")],
			vec![cell(t!("k_quotes"), DIM), cell(onoff(c.quotes_on()), "")],
		];
		let mut b = String::new();
		tbl(&mut b, &["".into(), "".into()], &rows);
		print!("{b}");
		eprintln!("{}", ce(DIM, CFG_USAGE));
		return;
	}
	let (cat, args) = (args[0].as_str(), &args[1..]);
	match cat {
		"hosts" => hosts(c, args),
		"quotes" => {
			let Some(a) = args.first() else {
				println!("{}: {}", t!("k_quotes"), co(CYN, &onoff(c.quotes_on())));
				return;
			};
			if a != "on" && a != "off" {
				die(&format!("{} sio config quotes on|off", t!("usage")));
			}
			c.quotes = Some(a == "on");
			save(c);
			ok(&t!("quotes_set", co(CYN, &onoff(a == "on"))));
		}
		"lang" => {
			let ls = lang::langs();
			let Some(a) = args.first() else {
				let (cur, w) = (lang::code(&c.lang), ls.iter().map(|l| l.len()).max().unwrap_or(0));
				for l in &ls {
					println!("{} {} {}", mark(*l == cur), co(CYN, &format!("{l:w$}")), lang::name(l));
				}
				lang_env();
				return;
			};
			if a == "auto" {
				c.lang.clear();
			} else if !ls.contains(a) {
				die(&t!("unk_lang", ce(CYN, a), ce(CYN, "sio config lang")));
			} else {
				c.lang = a.clone();
			}
			save(c);
			let lc = lang::code(&c.lang);
			lang::load(&lc); // confirm in the new lang
			ok(&t!("lang_set", co(CYN, &lc), lang::name(&lc)));
			lang_env();
		}
		_ => die(&format!("{}\n{CFG_USAGE}", t!("unk_cmd", ce(CYN, &format!("config {cat}"))))),
	}
}

fn hosts(c: &mut Cfg, args: &[String]) {
	let Some(sub) = args.first() else {
		if c.hosts.is_empty() {
			warn(&t!("no_hosts", ce(CYN, "sio config hosts add")));
		}
		let w = c.hosts.keys().map(|k| k.len()).max().unwrap_or(0);
		for (k, h) in &c.hosts {
			let li = if h.session.is_empty() { String::new() } else { co(DIM, &format!(" ({})", t!("logged_in"))) };
			println!("{} {} {}{li}", mark(*k == c.main), co(CYN, &format!("{k:w$}")), co(DIM, &h.url));
		}
		return;
	};
	let args = &args[1..];
	match sub.as_str() {
		"add" => {
			let mk = args.iter().any(|a| a == "--main" || a == "-m");
			let r: Vec<String> = args.iter().filter(|a| *a != "--main" && *a != "-m").cloned().collect();
			need(&r, 2, "config hosts add [--main] <name> <domain> [token]");
			let mut u = r[1].trim_end_matches('/').to_string();
			if !u.contains("://") {
				u = format!("https://{u}");
			}
			let tok = if r.len() > 2 { r[2].clone() } else { ask_token(&u) };
			let h = Host { url: u, token: tok, session: String::new() };
			c.hosts.insert(r[0].clone(), h.clone());
			if mk || c.main.is_empty() {
				c.main = r[0].clone();
			}
			save(c);
			match spin::wait(t!("w_ping", r[0]), || h.ping()) {
				Ok(who) => ok(&t!("added", co(CYN, &r[0]), co(CYN, &who))),
				Err(Err::Html) => warn(&t!("added_noauth", ce(CYN, &r[0]), t!("not_api", r[0]))),
				Err(e) => warn(&t!("added_noauth", ce(CYN, &r[0]), e)),
			}
		}
		"rm" => {
			need(args, 1, "config hosts rm <name>");
			let n = hname(c, &args[0]);
			c.hosts.remove(&n);
			if c.main == n {
				c.main.clear();
			}
			let _ = c.save();
			ok(&t!("removed", co(CYN, &n)));
		}
		"main" => {
			need(args, 1, "config hosts main <name>");
			c.main = hname(c, &args[0]);
			let _ = c.save();
			ok(&t!("main_set", co(CYN, &args[0])));
		}
		"token" => {
			need(args, 1, "config hosts token <name> [token]");
			let n = hname(c, &args[0]);
			let tok = if args.len() > 1 { args[1].clone() } else { ask_token(&c.hosts[&n].url) };
			c.hosts.get_mut(&n).unwrap().token = tok;
			let _ = c.save();
			ok(&t!("tok_set", co(CYN, &n)));
		}
		"login" => {
			need(args, 1, "config hosts login <name> [user]");
			let n = hname(c, &args[0]);
			let u = match args.get(1) {
				Some(u) => u.clone(),
				None => {
					eprint!("{}{}", ce(CYN, "? "), t!("login_user", ce(CYN, &c.hosts[&n].url)));
					let _ = std::io::stderr().flush();
					line()
				}
			};
			eprint!("{}{}", ce(CYN, "? "), t!("login_pass"));
			let _ = std::io::stderr().flush();
			let pw = rpassword::read_password().unwrap_or_else(|e| die(&e.to_string())); // no echo
			let mut h = c.hosts[&n].clone();
			match spin::wait(t!("w_login", n), || h.login(&u, &pw)) {
				Ok(()) => {}
				Err(Err::Login) => die(&t!("login_bad", ce(CYN, &n))),
				Err(Err::Tfa) => die(&t!("login_2fa", ce(CYN, &n))),
				Err(e) => fail(&n, &e),
			}
			c.hosts.insert(n.clone(), h);
			save(c);
			ok(&t!("login_ok", co(CYN, &n), co(CYN, &u)));
		}
		"logout" => {
			need(args, 1, "config hosts logout <name>");
			let n = hname(c, &args[0]);
			c.hosts.get_mut(&n).unwrap().session.clear();
			let _ = c.save();
			ok(&t!("logout_ok", co(CYN, &n)));
		}
		_ => die(&format!("{}\n{CFG_USAGE}", t!("unk_cmd", ce(CYN, &format!("config hosts {sub}"))))),
	}
}
