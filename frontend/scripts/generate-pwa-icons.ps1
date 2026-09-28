Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

Add-Type -AssemblyName System.Drawing

$root = Split-Path -Parent $PSScriptRoot
$out = Join-Path $root "public\icons\pwa"

New-Item `
    -ItemType Directory `
    -Force `
    -Path $out |
    Out-Null

$red =
    [System.Drawing.Color]::
    FromArgb(
        230,
        0,
        35
    )

$black =
    [System.Drawing.Color]::
    FromArgb(
        17,
        17,
        19
    )

$white =
    [System.Drawing.Color]::
    White

function New-EneIcon {
    param(
        [int]$Size,
        [int]$SymbolSize,
        [string]$Path,
        [bool]$Maskable = $false
    )

    $bitmap =
        New-Object `
            System.Drawing.Bitmap(
                $Size,
                $Size
            )

    $graphics =
        [System.Drawing.Graphics]::
        FromImage(
            $bitmap
        )

    $graphics.SmoothingMode =
        [System.Drawing.Drawing2D.SmoothingMode]::
        AntiAlias

    $graphics.PixelOffsetMode =
        [System.Drawing.Drawing2D.PixelOffsetMode]::
        HighQuality

    if ($Maskable) {
        $graphics.Clear(
            $red
        )
    }
    else {
        $graphics.Clear(
            $white
        )
    }

    $scale =
        [double]$SymbolSize /
        180.0

    $offsetX =
        (
            [double]$Size -
            $SymbolSize
        ) /
        2.0

    $offsetY =
        (
            [double]$Size -
            $SymbolSize
        ) /
        2.0

    function Pt {
        param(
            [double]$X,
            [double]$Y
        )

        return `
            [System.Drawing.PointF]::
            new(
                [single](
                    $offsetX +
                    $X *
                    $scale
                ),

                [single](
                    $offsetY +
                    $Y *
                    $scale
                )
            )
    }

    $redBrush =
        New-Object `
            System.Drawing.SolidBrush(
                $red
            )

    $whiteBrush =
        New-Object `
            System.Drawing.SolidBrush(
                $white
            )

    $pen =
        New-Object `
            System.Drawing.Pen(
                $black,
                [single](
                    14 *
                    $scale
                )
            )

    $curve =
        New-Object `
            System.Drawing.Drawing2D.GraphicsPath

    try {
        $graphics.FillEllipse(
            $redBrush,

            [single](
                $offsetX +
                8 *
                $scale
            ),

            [single](
                $offsetY +
                8 *
                $scale
            ),

            [single](
                164 *
                $scale
            ),

            [single](
                164 *
                $scale
            )
        )

        $e =
            [System.Drawing.PointF[]]@(
                (Pt 51 51),
                (Pt 99 51),
                (Pt 99 65),
                (Pt 67 65),
                (Pt 67 81),
                (Pt 94 81),
                (Pt 94 95),
                (Pt 67 95),
                (Pt 67 111),
                (Pt 101 111),
                (Pt 101 125),
                (Pt 51 125)
            )

        $graphics.FillPolygon(
            $whiteBrush,
            $e
        )

        $curve.AddBezier(
            (Pt 96 61),
            (Pt 126 61),
            (Pt 143 72),
            (Pt 149 89)
        )

        $curve.AddBezier(
            (Pt 149 89),
            (Pt 155 106),
            (Pt 150 122),
            (Pt 139 134)
        )

        $curve.AddBezier(
            (Pt 139 134),
            (Pt 128 146),
            (Pt 114 152),
            (Pt 96 152)
        )

        $pen.StartCap =
            [System.Drawing.Drawing2D.LineCap]::
            Round

        $pen.EndCap =
            [System.Drawing.Drawing2D.LineCap]::
            Round

        $pen.LineJoin =
            [System.Drawing.Drawing2D.LineJoin]::
            Round

        $graphics.DrawPath(
            $pen,
            $curve
        )

        $bitmap.Save(
            $Path,

            [System.Drawing.Imaging.ImageFormat]::
            Png
        )
    }
    finally {
        $curve.Dispose()
        $pen.Dispose()
        $redBrush.Dispose()
        $whiteBrush.Dispose()
        $graphics.Dispose()
        $bitmap.Dispose()
    }
}

New-EneIcon `
    -Size 192 `
    -SymbolSize 164 `
    -Path (
        Join-Path `
            $out `
            "icon-192.png"
    )

New-EneIcon `
    -Size 512 `
    -SymbolSize 432 `
    -Path (
        Join-Path `
            $out `
            "icon-512.png"
    )

New-EneIcon `
    -Size 180 `
    -SymbolSize 154 `
    -Path (
        Join-Path `
            $out `
            "apple-touch-icon.png"
    )

New-EneIcon `
    -Size 512 `
    -SymbolSize 350 `
    -Path (
        Join-Path `
            $out `
            "maskable-512.png"
    ) `
    -Maskable $true

Write-Host `
    "PWA icons created in public\icons\pwa" `
    -ForegroundColor Green