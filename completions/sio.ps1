Register-ArgumentCompleter -Native -CommandName sio -ScriptBlock {
	param($word, $ast, $pos)
	$w = @($ast.CommandElements | % { "$_" })
	if ($word -ne '') { $w = $w[0..($w.Count - 2)] } # drop the word being typed
	$s = ($w | Select-Object -Skip 1) -join ' '
	$c = switch -regex ($s) {
		'^$' { 'config', 'ping', 'info', 'tree', 'submit', 'probs', 'subs' }
		'^config$' { 'hosts', 'lang', 'quotes' }
		'^config quotes$' { 'on', 'off' }
		'^config hosts$' { 'add', 'rm', 'main', 'token', 'login', 'logout' }
		'^config hosts add$' { '--main' }
		'^(config hosts (rm|main|token|login|logout)|ping|info|tree)$' { sio config hosts 2>$null | % { $_.Substring(2).Split(' ')[0] } }
		'^config lang$' { @(sio config lang 2>$null | % { $_.Substring(2).Split(' ')[0] }) + 'auto' }
		'^(submit|sub) ' { Get-ChildItem -Name "$word*" }
	}
	$c | ? { $_ -like "$word*" } | % { [System.Management.Automation.CompletionResult]::new($_) }
}
