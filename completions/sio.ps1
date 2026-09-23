Register-ArgumentCompleter -Native -CommandName sio -ScriptBlock {
	param($word, $ast, $pos)
	$w = @($ast.CommandElements | % { "$_" })
	if ($word -ne '') { $w = $w[0..($w.Count - 2)] } # drop the word being typed
	$s = ($w | Select-Object -Skip 1) -join ' '
	$c = switch -regex ($s) {
		'^$' { 'config', 'ping', 'info', 'tree', 'upload', 'probs', 'subs' }
		'^config$' { 'hosts', 'lang', 'quotes' }
		'^config quotes$' { 'on', 'off' }
		'^config hosts$' { 'add', 'rm', 'main', 'token' }
		'^config hosts add$' { '--main' }
		'^(config hosts (rm|main|token)|ping|info|tree)$' { sio config hosts 2>$null | % { $_.Substring(2).Split(' ')[0] } }
		'^config lang$' { @(sio config lang 2>$null | % { $_.Substring(2).Split(' ')[0] }) + 'auto' }
		'^(upload|up) ' { Get-ChildItem -Name "$word*" }
	}
	$c | ? { $_ -like "$word*" } | % { [System.Management.Automation.CompletionResult]::new($_) }
}
