use crate::cfg::Cfg;
use crate::out::{CYN, DIM, GRN, GRY, ce, cell, co, die, link, ok, page, tbl, warn};
use crate::{fail, host, spin::wait, t};

pub fn ping(c: &Cfg, args: &[String]) {
	let (n, h) = host(c, args.first().map_or("", |s| s));
	let who = wait(t!("w_ping", n), || h.ping()).unwrap_or_else(|e| fail(&n, &e));
	ok(&t!("ping_ok", co(CYN, &n), co(CYN, &who)));
}

pub fn info(c: &Cfg, args: &[String]) {
	let (n, h) = host(c, args.first().map_or("", |s| s));
	let who = wait(t!("w_ping", n), || h.ping()).unwrap_or_else(|e| fail(&n, &e));
	let hd = format!("{} {} {}", co(GRN, "✓"), t!("ping_ok", co(CYN, &n), co(CYN, &who)), co(DIM, &h.url));
	let cs = match wait(t!("w_contests", n), || h.contests()) {
		Ok(cs) => cs,
		Err(e) => {
			println!("{hd}");
			if e.code() == 404 {
				die(&format!("{e}\n  {}", ce(GRY, &link(*crate::out::COL_ERR, "https://pastebin.com/chMT18MG", &t!("why"))))); // old oioioi, no contest_list
			}
			fail(&n, &e)
		}
	};
	if cs.is_empty() {
		warn(&t!("no_contests", ce(CYN, &n)));
	}
	let rows: Vec<_> = cs.iter().map(|ct| vec![cell(&ct.id, CYN), cell(&ct.name, "")]).collect();
	let mut b = String::new();
	tbl(&mut b, &[t!("k_contest"), t!("k_name")], &rows);
	page(&hd, &b);
}
