mod api;
mod cfg;
mod cmd;
mod lang;
mod out;
mod quote;
mod spin;
mod tui;
mod web;

use api::Err;
use cfg::{Cfg, Host};
use out::{BOLD, CYN, ce, die};
use std::io::{BufRead, IsTerminal, Write};

pub const CFG_USAGE: &str = "  sio config [hosts|lang|quotes]
  sio config hosts [add|rm|main|token|login|logout]
  sio config hosts add [--main] <name> <domain> [token]
  sio config hosts rm <name>
  sio config hosts main <name>
  sio config hosts token <name> [token]
  sio config hosts login <name> [user]
  sio config hosts logout <name>
  sio config lang [code|auto]
  sio config quotes [on|off]";

const USAGE: &str = "
  sio ping [name]
  sio info [name]
  sio tree [name]
  sio submit [-n] <[host/]contest> <prob|file> [file]
  sio probs <[host/]contest>
  sio subs <[host/]contest> [prob]";

pub fn usage() -> String {
	format!("{CFG_USAGE}{USAGE}")
}

pub fn need(args: &[String], n: usize, u: &str) {
	if args.len() < n {
		die(&format!("{} sio {u}", t!("usage")));
	}
}

pub fn hname(c: &Cfg, name: &str) -> String {
	let n = if name.is_empty() { c.main.as_str() } else { name }; // "" = main, dies if unknown
	if n.is_empty() {
		die(&t!("no_main", ce(CYN, "sio config hosts main <name>")));
	}
	if !c.hosts.contains_key(n) {
		die(&t!("unk_host", ce(CYN, n)));
	}
	n.into()
}

pub fn host(c: &Cfg, name: &str) -> (String, Host) {
	let n = hname(c, name);
	let h = c.hosts[&n].clone();
	(n, h)
}

pub fn hct(c: &Cfg, arg: &str) -> (String, Host, String) {
	let (hn, ct) = arg.split_once('/').unwrap_or(("", arg)); // [host/]contest -> host name, host, contest
	let (hn, h) = host(c, hn);
	(hn, h, ct.into())
}

pub fn line() -> String {
	let mut s = String::new();
	let _ = std::io::stdin().lock().read_line(&mut s);
	s.trim().into()
}

pub fn ask_token(url: &str) -> String {
	eprint!("{}{}", ce(CYN, "? "), t!("tok_ask", ce(CYN, &format!("{url}/api/token"))));
	let _ = std::io::stderr().flush();
	line()
}

pub fn fail(hn: &str, e: &Err) -> ! {
	match e {
		Err::Html => die(&t!("not_api", ce(CYN, hn))),
		e if e.code() == 401 => die(&t!("bad_tok", ce(CYN, hn), ce(CYN, &format!("sio config hosts token {hn}")))),
		e => die(&e.to_string()),
	}
}

fn main() {
	#[cfg(windows)]
	let _ = crossterm::ansi_support::supports_ansi(); // turns vt mode on, old conhost has it off
	let mut c = Cfg::load();
	lang::load(&lang::code(&c.lang));
	let a: Vec<String> = std::env::args().skip(1).collect();
	if a.is_empty() {
		eprintln!("{}\n{}", ce(BOLD, &t!("usage")), usage());
		std::process::exit(1);
	}
	let (cm, args) = (a[0].as_str(), &a[1..]);
	match cm {
		"config" | "cfg" => cmd::cfg::run(&mut c, args),
		"ping" => cmd::info::ping(&c, args),
		"info" => cmd::info::info(&c, args),
		"tree" => cmd::tree::run(&c, args),
		"submit" | "sub" => cmd::up::run(&c, args),
		"probs" => cmd::probs::probs_cmd(&c, args),
		"subs" => cmd::probs::subs_cmd(&c, args),
		_ => die(&format!("{}\n{}", t!("unk_cmd", ce(CYN, cm)), usage())),
	}
	if c.quotes_on() && std::io::stdout().is_terminal() {
		out::quote(); // never when piped
	}
}
