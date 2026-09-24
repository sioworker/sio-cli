-- pkgit -i https://github.com/sioworker/sio-cli
-- needs go 1.27+ and make on the system, not built from git here

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

local function go_ok()
	local v = sh("go version")
	local ma, mi = v:match("go(%d+)%.(%d+)")
	ma, mi = tonumber(ma), tonumber(mi)
	if ma and (ma > 1 or mi >= 27) then return true end
	if ma and mi >= 21 and sh("go env GOTOOLCHAIN") ~= "local" then return true end
	local got = ma and ("found go" .. ma .. "." .. mi .. (mi >= 21 and " with GOTOOLCHAIN=local" or "")) or "no go found"
	io.stderr:write("sio needs go 1.27+ to build (" .. got .. ")\ninstall it first: your distros go package or https://go.dev/dl\n")
	return false
end

targets = {
	default = {
		build = function()
			if not go_ok() then return 1 end
			return os.execute("make")
		end,
		install = function()
			return os.execute("make install PREFIX=" .. prefix)
		end,
		uninstall = function()
			return os.execute("make uninstall PREFIX=" .. prefix)
		end,
	},
	quiet = {
		build = function()
			if not go_ok() then return 1 end
			return os.execute("make >/tmp/sio_build.log 2>&1")
		end,
		install = function()
			return os.execute("make install PREFIX=" .. prefix .. " >/tmp/sio_build.log 2>&1")
		end,
		uninstall = function()
			return os.execute("make uninstall PREFIX=" .. prefix .. " >/tmp/sio_build.log 2>&1")
		end,
	},
}
