use crate::api::{Prob, Sub};
use crate::cfg::{Cfg, Host};
use crate::out::{CYN, Cell, DIM, ce, cell, page, scell, score, stat, tbl, warn};
use crate::{fail, hct, need, spin, t};
use regex::Regex;
use std::sync::LazyLock;

pub fn fetch(h: &Host, ct: &str) -> (crate::api::Res<Vec<Prob>>, bool) {
	match h.probs(ct) {
		Err(e) if e.code() / 100 == 5 => {
			let r = h.probs_web(ct); // problem_list 500s on some contests, fall back to the web page
			let w = r.is_ok(); // bool = came from the web page
			(r, w)
		}
		r => (r, false),
	}
}

pub fn probs(hn: &str, h: &Host, ct: &str) -> Vec<Prob> {
	let (r, web) = spin::wait(t!("w_probs", format!("{hn}/{ct}")), || fetch(h, ct));
	if web {
		warn(&t!("probs_web", ce(CYN, &format!("{hn}/{ct}"))));
	}
	r.unwrap_or_else(|e| fail(hn, &e))
}

static TEX: LazyLock<Regex> = LazyLock::new(|| Regex::new(r"\\[a-zA-Z]+\{([^{}]*)\}").unwrap());
static TCMD: LazyLock<Regex> = LazyLock::new(|| Regex::new(r"\\[a-zA-Z]+ ?").unwrap());

pub fn tex(s: &str) -> String {
	let mut s = s.to_string(); // $k$-inwersje, \mbox{x} -> k-inwersje, x
	while TEX.is_match(&s) {
		s = TEX.replace_all(&s, "$1").into();
	}
	TCMD.replace_all(&s.replace('$', ""), "").trim().into()
}

pub fn probs_cmd(c: &Cfg, args: &[String]) {
	need(args, 1, "probs <[host/]contest>");
	let (hn, h, ct) = hct(c, &args[0]);
	let ps = probs(&hn, &h, &ct);
	if ps.is_empty() {
		warn(&t!("no_probs", ce(CYN, &format!("{hn}/{ct}"))));
	}
	let rows: Vec<Vec<Cell>> = ps
		.iter()
		.map(|p| {
			let (st, sc) = match &p.res {
				Some(r) if !r.status.is_empty() => (stat(&r.status), score(&r.score)),
				_ => (cell("-", DIM), "-".into()),
			};
			let l = p.left.map_or("-".into(), |l| l.to_string());
			vec![cell(&p.short, CYN), cell(tex(&p.name), ""), scell(&sc), cell(l, DIM), st]
		})
		.collect();
	let mut b = String::new();
	tbl(&mut b, &[t!("k_prob"), t!("k_name"), t!("k_score"), t!("k_left"), t!("k_stat")], &rows);
	page("", &b);
}

pub fn subs_cmd(c: &Cfg, args: &[String]) {
	need(args, 1, "subs <[host/]contest> [prob]");
	let (hn, h, ct) = hct(c, &args[0]);
	let pn: Vec<String> = if args.len() > 1 { args[1..].to_vec() } else { probs(&hn, &h, &ct).into_iter().map(|p| p.short).collect() };
	let mut ss: Vec<Sub> = vec![];
	for (i, p) in pn.iter().enumerate() {
		let (s, tr) = spin::wait(t!("w_subs", p, format!("{}/{}", i + 1, pn.len())), || h.subs(&ct, p)).unwrap_or_else(|e| fail(&hn, &e));
		if tr {
			warn(&t!("trunc", ce(CYN, p)));
		}
		ss.extend(s);
	}
	if ss.is_empty() {
		warn(&t!("no_subs", ce(CYN, &format!("{hn}/{ct}"))));
	}
	ss.sort_by_key(|s| std::cmp::Reverse(s.date)); // newest first
	let rows: Vec<Vec<Cell>> = ss
		.iter()
		.map(|s| {
			let d = s.date.map_or(String::new(), |d| d.with_timezone(&chrono::Local).format("%Y-%m-%d %H:%M").to_string());
			vec![cell(s.id.to_string(), DIM), cell(&s.prob, CYN), cell(d, ""), scell(&s.score.map_or("-".into(), |x| x.to_string())), stat(&s.status)]
		})
		.collect();
	let mut b = String::new();
	tbl(&mut b, &[t!("k_id"), t!("k_prob"), t!("k_date"), t!("k_score"), t!("k_stat")], &rows);
	page("", &b);
}
