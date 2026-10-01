-- pkgit -i https://github.com/sioworker/sio-cli
-- needs rust 1.88+ (cargo) on the system, not built from git here

bldit_version   = "1.2.0"
package_version = "0.1.0"

dependencies = {}

local function sh(c)
	local p = io.popen(c .. " 2>/dev/null")
	if not p then return "" end
	local l = p:read("*l") or ""
	p:close()
	return l
end

local function rust_ok()
	local v = sh("rustc --version")
	local ma, mi = v:match("rustc (%d+)%.(%d+)")
	ma, mi = tonumber(ma), tonumber(mi)
	local cg = sh("cargo --version") ~= ""
	if ma and cg and (ma > 1 or mi >= 88) then return true end
	local got = not ma and "no rust found" or not cg and "no cargo found" or ("found rustc " .. ma .. "." .. mi)
	io.stderr:write("sio needs rust 1.88+ (rustc + cargo) to build (" .. got .. ")\ninstall it first: https://rustup.rs or your distros rust package\n")
	return false
end

local function q(s) return "'" .. s:gsub("'", "'\\''") .. "'" end -- sh quote

local function put(un) -- cp or rm the bin + completions under prefix, user prefix keeps fish in ~/.config like before
	local h, b, sh_ = os.getenv("HOME") or "", prefix .. "/bin", prefix .. "/share"
	local fish = prefix == h .. "/.local" and h .. "/.config/fish/completions" or sh_ .. "/fish/vendor_completions.d"
	local f = {
		{ "target/release/sio", b, "sio" },
		{ "completions/sio.fish", fish, "sio.fish" },
		{ "completions/sio.bash", sh_ .. "/bash-completion/completions", "sio" },
		{ "completions/_sio", sh_ .. "/zsh/site-functions", "_sio" },
	}
	local c = {}
	for _, x in ipairs(f) do
		c[#c + 1] = un and ("rm -f " .. q(x[2] .. "/" .. x[3])) or ("mkdir -p " .. q(x[2]) .. " && cp " .. q(x[1]) .. " " .. q(x[2] .. "/" .. x[3]))
	end
	return table.concat(c, " && ")
end

local log = " >/tmp/sio_build.log 2>&1"

targets = {
	default = {
		build = function()
			if not rust_ok() then return 1 end
			return os.execute("cargo build --release --locked")
		end,
		install = function()
			return os.execute(put(false))
		end,
		uninstall = function()
			return os.execute(put(true))
		end,
	},
	quiet = {
		build = function()
			if not rust_ok() then return 1 end
			return os.execute("cargo build --release --locked" .. log)
		end,
		install = function()
			return os.execute("{ " .. put(false) .. "; }" .. log)
		end,
		uninstall = function()
			return os.execute("{ " .. put(true) .. "; }" .. log)
		end,
	},
}
