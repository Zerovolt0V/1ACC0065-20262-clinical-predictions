# run_bench.ps1: benchmark secuencial vs concurrente, un proceso independiente por número de workers.
# Uso: powershell -ExecutionPolicy Bypass -File bench/run_bench.ps1 [-Workers 1,2,4,8] [-SinSecuencial] [-Epocas 3] [-Repeticiones 11]
param(
    [int]$Epocas = 3,
    [int]$Repeticiones = 11,
    [string]$Workers = "1,2,4,8,12,16,24",
    [switch]$SinSecuencial,
    [string]$Salida = "bench/results.csv"
)
$ErrorActionPreference = "Stop"
$listaW = $Workers -split "," | ForEach-Object { [int]$_.Trim() }
$raiz = Split-Path -Parent (Split-Path -Parent $MyInvocation.MyCommand.Path)
Set-Location $raiz
New-Item -ItemType Directory -Force bench/bin, bench/salidas | Out-Null

Write-Host "Compilando..."
go build -o bench/bin/secuencial.exe secuencial/secuencial.go
go build -o bench/bin/concurrente.exe concurrente/concurrente.go
$cpus = [Environment]::ProcessorCount

if (-not (Test-Path $Salida) -or -not $SinSecuencial) {
    "modo,workers,epocas,repeticion,tiempo_ms,cpu_ms_entrenamiento,wall_ms_entrenamiento,pico_mb,cpus" | Set-Content $Salida
}

function Interpolar($muestras, $t) {
    if ($muestras.Count -eq 0) { return 0 }
    if ($t -le $muestras[0].t) { return $muestras[0].cpu }
    for ($i = 1; $i -lt $muestras.Count; $i++) {
        if ($t -le $muestras[$i].t) {
            $a = $muestras[$i - 1]; $b = $muestras[$i]
            if ($b.t -eq $a.t) { return $b.cpu }
            return $a.cpu + ($b.cpu - $a.cpu) * ($t - $a.t) / ($b.t - $a.t)
        }
    }
    return $muestras[$muestras.Count - 1].cpu
}

function Ejecutar($modo, $w, $exe, $argumentos) {
    $log = "bench/salidas/${modo}_w${w}.txt"
    Write-Host ("[{0}] {1}  workers={2}  ({3} epocas x {4} repeticiones)" -f (Get-Date -Format "HH:mm:ss"), $modo, $w, $Epocas, $Repeticiones)
    $psi = New-Object System.Diagnostics.ProcessStartInfo
    $psi.FileName = (Resolve-Path $exe).Path
    $psi.Arguments = $argumentos
    $psi.WorkingDirectory = $raiz
    $psi.UseShellExecute = $false
    $psi.RedirectStandardOutput = $true
    $psi.StandardOutputEncoding = [System.Text.Encoding]::UTF8
    $p = [System.Diagnostics.Process]::Start($psi)
    $lectura = $p.StandardOutput.ReadToEndAsync()

    $muestras = New-Object System.Collections.Generic.List[object]
    $pico = 0
    while (-not $p.HasExited) {
        try {
            $p.Refresh()
            $muestras.Add([pscustomobject]@{
                t   = [DateTimeOffset]::UtcNow.ToUnixTimeMilliseconds()
                cpu = $p.TotalProcessorTime.TotalMilliseconds
            })
            if ($p.WorkingSet64 -gt $pico) { $pico = $p.WorkingSet64 }
        } catch {}
        Start-Sleep -Milliseconds 100
    }
    $texto = $lectura.Result
    $texto | Set-Content -Encoding UTF8 $log

    $ini = [int64]([regex]::Match($texto, 'MARCA_INICIO_MS=(\d+)').Groups[1].Value)
    $fin = [int64]([regex]::Match($texto, 'MARCA_FIN_MS=(\d+)').Groups[1].Value)
    $cpuEnt = [math]::Round((Interpolar $muestras $fin) - (Interpolar $muestras $ini))
    $wallEnt = $fin - $ini
    $picoMb = [math]::Round($pico / 1MB, 1)
    $tiempos = [regex]::Matches($texto, 'TIEMPO_MS=(\d+)') | ForEach-Object { [int]$_.Groups[1].Value }
    $r = 0
    foreach ($t in $tiempos) {
        $r++
        "$modo,$w,$Epocas,$r,$t,$cpuEnt,$wallEnt,$picoMb,$cpus" | Add-Content $Salida
    }
    $medidas = $tiempos | Select-Object -Skip 1
    $media = ($medidas | Measure-Object -Average).Average
    Write-Host ("           media (sin calentamiento) = {0:N0} ms | CPU en entrenamiento = {1:N0} ms de {2:N0} ms | pico RAM = {3} MB" -f $media, $cpuEnt, $wallEnt, $picoMb)
}

$inicioTotal = Get-Date
if (-not $SinSecuencial) {
    Ejecutar "secuencial" 0 "bench/bin/secuencial.exe" "$Epocas $Repeticiones"
}
foreach ($w in $listaW) {
    Ejecutar "concurrente" $w "bench/bin/concurrente.exe" "$w $Epocas $Repeticiones"
}
Write-Host ("Listo en {0:N1} min. Resultados en {1}" -f ((Get-Date) - $inicioTotal).TotalMinutes, $Salida)

