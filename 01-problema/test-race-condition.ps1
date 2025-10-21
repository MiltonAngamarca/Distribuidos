# Script PowerShell para demostrar race conditions en el sistema de reservas
# Este script envía múltiples peticiones concurrentes al mismo asiento
# para provocar el problema de doble reserva

Write-Host "🎬 DEMOSTRACIÓN DE RACE CONDITIONS" -ForegroundColor Cyan
Write-Host "==================================" -ForegroundColor Cyan
Write-Host ""

# URLs de los servidores
$LOAD_BALANCER = "http://localhost:8080"
$SERVIDOR_1 = "http://localhost:8081"
$SERVIDOR_2 = "http://localhost:8082"
$SERVIDOR_3 = "http://localhost:8083"

# Función para verificar si los servidores están activos
function Check-Servers {
    Write-Host "🔍 Verificando estado de los servidores..." -ForegroundColor Blue
    
    for ($i = 1; $i -le 3; $i++) {
        $url = "http://localhost:808$i/health"
        try {
            $response = Invoke-RestMethod -Uri $url -Method Get -TimeoutSec 5
            Write-Host "✅ Servidor $i (puerto 808$i) - ACTIVO" -ForegroundColor Green
        }
        catch {
            Write-Host "❌ Servidor $i (puerto 808$i) - INACTIVO" -ForegroundColor Red
        }
    }
    Write-Host ""
}

# Función para resetear todos los servidores
function Reset-Servers {
    Write-Host "🔄 Reseteando todos los servidores..." -ForegroundColor Yellow
    
    for ($i = 1; $i -le 3; $i++) {
        $url = "http://localhost:808$i/reset"
        try {
            Invoke-RestMethod -Uri $url -Method Post -TimeoutSec 5 | Out-Null
        }
        catch {
            Write-Host "Error reseteando servidor $i" -ForegroundColor Red
        }
    }
    
    Write-Host "✅ Servidores reseteados" -ForegroundColor Green
    Write-Host ""
}

# Función para mostrar el estado de un asiento en todos los servidores
function Show-SeatStatus {
    param([int]$SeatNumber)
    
    Write-Host "📊 Estado del asiento $SeatNumber en todos los servidores:" -ForegroundColor Blue
    
    for ($i = 1; $i -le 3; $i++) {
        $url = "http://localhost:808$i/asiento/$SeatNumber"
        try {
            $response = Invoke-RestMethod -Uri $url -Method Get -TimeoutSec 5
            
            if ($response.asiento.disponible) {
                Write-Host "  Servidor $i: DISPONIBLE" -ForegroundColor Green
            }
            else {
                Write-Host "  Servidor $i: RESERVADO por '$($response.asiento.cliente)'" -ForegroundColor Red
            }
        }
        catch {
            Write-Host "  Servidor $i: ERROR" -ForegroundColor Red
        }
    }
    Write-Host ""
}

# Función para reservar asiento
function Reserve-Seat {
    param(
        [int]$SeatNumber,
        [string]$ClientName,
        [int]$ServerPort
    )
    
    $url = "http://localhost:$ServerPort/reservar"
    $body = @{
        numero = $SeatNumber
        cliente = $ClientName
    } | ConvertTo-Json
    
    try {
        $response = Invoke-RestMethod -Uri $url -Method Post -Body $body -ContentType "application/json" -TimeoutSec 5
        
        if ($response.success) {
            Write-Host "✅ [$ClientName] Reserva exitosa en servidor puerto $ServerPort" -ForegroundColor Green
            return $true
        }
        else {
            Write-Host "❌ [$ClientName] Error en servidor puerto $ServerPort: $($response.error)" -ForegroundColor Red
            return $false
        }
    }
    catch {
        Write-Host "❌ [$ClientName] Error de conexión en servidor puerto $ServerPort" -ForegroundColor Red
        return $false
    }
}

# Función principal para demostrar race condition
function Demonstrate-RaceCondition {
    param([int]$SeatNumber)
    
    Write-Host "🚀 INICIANDO PRUEBA DE RACE CONDITION" -ForegroundColor Yellow
    Write-Host "Asiento objetivo: $SeatNumber" -ForegroundColor Yellow
    Write-Host ""
    
    # Mostrar estado inicial
    Write-Host "📋 Estado inicial:" -ForegroundColor Blue
    Show-SeatStatus -SeatNumber $SeatNumber
    
    # Enviar peticiones concurrentes
    Write-Host "⚡ Enviando peticiones concurrentes..." -ForegroundColor Yellow
    Write-Host ""
    
    # Ejecutar reservas en paralelo usando Jobs
    $jobs = @()
    $jobs += Start-Job -ScriptBlock {
        param($SeatNumber, $ServerPort)
        
        $url = "http://localhost:$ServerPort/reservar"
        $body = @{
            numero = $SeatNumber
            cliente = "Cliente-A"
        } | ConvertTo-Json
        
        try {
            $response = Invoke-RestMethod -Uri $url -Method Post -Body $body -ContentType "application/json" -TimeoutSec 5
            return @{ Success = $response.success; Client = "Cliente-A"; Port = $ServerPort; Error = $response.error }
        }
        catch {
            return @{ Success = $false; Client = "Cliente-A"; Port = $ServerPort; Error = $_.Exception.Message }
        }
    } -ArgumentList $SeatNumber, 8081
    
    $jobs += Start-Job -ScriptBlock {
        param($SeatNumber, $ServerPort)
        
        $url = "http://localhost:$ServerPort/reservar"
        $body = @{
            numero = $SeatNumber
            cliente = "Cliente-B"
        } | ConvertTo-Json
        
        try {
            $response = Invoke-RestMethod -Uri $url -Method Post -Body $body -ContentType "application/json" -TimeoutSec 5
            return @{ Success = $response.success; Client = "Cliente-B"; Port = $ServerPort; Error = $response.error }
        }
        catch {
            return @{ Success = $false; Client = "Cliente-B"; Port = $ServerPort; Error = $_.Exception.Message }
        }
    } -ArgumentList $SeatNumber, 8082
    
    $jobs += Start-Job -ScriptBlock {
        param($SeatNumber, $ServerPort)
        
        $url = "http://localhost:$ServerPort/reservar"
        $body = @{
            numero = $SeatNumber
            cliente = "Cliente-C"
        } | ConvertTo-Json
        
        try {
            $response = Invoke-RestMethod -Uri $url -Method Post -Body $body -ContentType "application/json" -TimeoutSec 5
            return @{ Success = $response.success; Client = "Cliente-C"; Port = $ServerPort; Error = $response.error }
        }
        catch {
            return @{ Success = $false; Client = "Cliente-C"; Port = $ServerPort; Error = $_.Exception.Message }
        }
    } -ArgumentList $SeatNumber, 8083
    
    # Esperar a que terminen todos los jobs
    $results = $jobs | Wait-Job | Receive-Job
    $jobs | Remove-Job
    
    # Mostrar resultados
    foreach ($result in $results) {
        if ($result.Success) {
            Write-Host "✅ [$($result.Client)] Reserva exitosa en servidor puerto $($result.Port)" -ForegroundColor Green
        }
        else {
            Write-Host "❌ [$($result.Client)] Error en servidor puerto $($result.Port): $($result.Error)" -ForegroundColor Red
        }
    }
    
    Write-Host ""
    Write-Host "📋 Estado final:" -ForegroundColor Blue
    Show-SeatStatus -SeatNumber $SeatNumber
    
    # Verificar si hay inconsistencia
    Write-Host "🔍 Análisis de resultados:" -ForegroundColor Yellow
    
    $reservedCount = 0
    for ($i = 1; $i -le 3; $i++) {
        $url = "http://localhost:808$i/asiento/$SeatNumber"
        try {
            $response = Invoke-RestMethod -Uri $url -Method Get -TimeoutSec 5
            if (-not $response.asiento.disponible) {
                $reservedCount++
            }
        }
        catch {
            # Ignorar errores de conexión para el análisis
        }
    }
    
    if ($reservedCount -gt 1) {
        Write-Host "🚨 RACE CONDITION DETECTADO!" -ForegroundColor Red
        Write-Host "   El asiento $SeatNumber está reservado en $reservedCount servidores" -ForegroundColor Red
        Write-Host "   Esto demuestra el problema de concurrencia" -ForegroundColor Red
        return $true
    }
    elseif ($reservedCount -eq 1) {
        Write-Host "⚠️  Solo un servidor tiene el asiento reservado" -ForegroundColor Yellow
        Write-Host "   Puede que el race condition no se haya manifestado esta vez" -ForegroundColor Yellow
        Write-Host "   Intenta ejecutar el script varias veces" -ForegroundColor Yellow
        return $false
    }
    else {
        Write-Host "✅ Ningún servidor tiene el asiento reservado" -ForegroundColor Green
        Write-Host "   Todas las peticiones fallaron (posible, pero inusual)" -ForegroundColor Yellow
        return $false
    }
}

# Función para ejecutar múltiples pruebas
function Run-MultipleTests {
    param(
        [int]$Iterations,
        [int]$SeatNumber
    )
    
    Write-Host "🔄 Ejecutando $Iterations pruebas para el asiento $SeatNumber" -ForegroundColor Yellow
    Write-Host ""
    
    $raceConditionsDetected = 0
    
    for ($i = 1; $i -le $Iterations; $i++) {
        Write-Host "--- Prueba $i/$Iterations ---" -ForegroundColor Blue
        
        # Resetear servidores
        Reset-Servers
        Start-Sleep -Seconds 1
        
        # Ejecutar prueba
        $raceDetected = Demonstrate-RaceCondition -SeatNumber $SeatNumber
        
        if ($raceDetected) {
            $raceConditionsDetected++
        }
        
        Write-Host ""
        Start-Sleep -Seconds 2
    }
    
    Write-Host "📊 RESUMEN DE PRUEBAS:" -ForegroundColor Yellow
    Write-Host "   Total de pruebas: $Iterations"
    Write-Host "   Race conditions detectados: $raceConditionsDetected"
    Write-Host "   Porcentaje de éxito: $([math]::Round(($raceConditionsDetected * 100) / $Iterations, 2))%"
    Write-Host ""
}

# Función para mostrar todos los asientos
function Show-AllSeats {
    Write-Host "📊 Estado de todos los asientos:" -ForegroundColor Blue
    
    for ($i = 1; $i -le 3; $i++) {
        Write-Host "Servidor $i (puerto 808$i):" -ForegroundColor Yellow
        $url = "http://localhost:808$i/estado"
        try {
            $response = Invoke-RestMethod -Uri $url -Method Get -TimeoutSec 5
            Write-Host "  Total: $($response.total_asientos), Disponibles: $($response.disponibles), Reservados: $($response.reservados)"
        }
        catch {
            Write-Host "  ERROR al conectar" -ForegroundColor Red
        }
    }
    Write-Host ""
}

# Menú principal
function Show-Menu {
    Write-Host "🎯 OPCIONES DISPONIBLES:" -ForegroundColor Blue
    Write-Host "1. Verificar estado de servidores"
    Write-Host "2. Resetear todos los servidores"
    Write-Host "3. Demostrar race condition (asiento específico)"
    Write-Host "4. Ejecutar múltiples pruebas"
    Write-Host "5. Mostrar estado de todos los asientos"
    Write-Host "6. Salir"
    Write-Host ""
}

# Script principal
function Main {
    Write-Host "🎬 SISTEMA DE DEMOSTRACIÓN DE RACE CONDITIONS" -ForegroundColor Green
    Write-Host "=============================================" -ForegroundColor Green
    Write-Host ""
    Write-Host "Este script demuestra race conditions en un sistema distribuido"
    Write-Host "donde múltiples servidores manejan el mismo estado sin sincronización."
    Write-Host ""
    
    # Verificar servidores inicialmente
    Check-Servers
    
    # Menú interactivo
    while ($true) {
        Show-Menu
        $choice = Read-Host "Selecciona una opción (1-6)"
        
        switch ($choice) {
            "1" {
                Check-Servers
            }
            "2" {
                Reset-Servers
            }
            "3" {
                $seat = Read-Host "Ingresa el número de asiento (1-50)"
                if ($seat -match '^\d+$' -and [int]$seat -ge 1 -and [int]$seat -le 50) {
                    Demonstrate-RaceCondition -SeatNumber ([int]$seat) | Out-Null
                }
                else {
                    Write-Host "❌ Número de asiento inválido" -ForegroundColor Red
                }
            }
            "4" {
                $iterations = Read-Host "Número de pruebas a ejecutar"
                $seat = Read-Host "Número de asiento (1-50)"
                if ($iterations -match '^\d+$' -and $seat -match '^\d+$' -and [int]$seat -ge 1 -and [int]$seat -le 50) {
                    Run-MultipleTests -Iterations ([int]$iterations) -SeatNumber ([int]$seat)
                }
                else {
                    Write-Host "❌ Valores inválidos" -ForegroundColor Red
                }
            }
            "5" {
                Show-AllSeats
            }
            "6" {
                Write-Host "👋 ¡Hasta luego!" -ForegroundColor Green
                return
            }
            default {
                Write-Host "❌ Opción inválida" -ForegroundColor Red
            }
        }
        
        Write-Host ""
        Read-Host "Presiona Enter para continuar..." | Out-Null
        Write-Host ""
    }
}

# Ejecutar script principal
Main