#Requires -Version 5.1
<#
.SYNOPSIS
  Render Mermaid sources in docs/diagrams/*.mmd to PNG.
.EXAMPLE
  .\scripts\render-diagrams.ps1
#>
$ErrorActionPreference = "Stop"
Set-Location (Split-Path -Parent $PSScriptRoot)

$diagramDir = Join-Path $PWD "docs\diagrams"
$mmdc = "npx"
$mmdcArgs = @("-y", "@mermaid-js/mermaid-cli@11.4.0")

Get-ChildItem $diagramDir -Filter "*.mmd" | ForEach-Object {
    $out = Join-Path $diagramDir ($_.BaseName + ".png")
    Write-Host "Rendering $($_.Name) -> $($_.BaseName).png"
    & $mmdc @mmdcArgs -i $_.FullName -o $out -b transparent -w 1400 -H 900
}

Write-Host "Done. PNG files in docs/diagrams/"
