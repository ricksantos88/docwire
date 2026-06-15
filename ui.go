package docwire

import (
	"fmt"
)

// DefaultSwaggerUIHTML retorna o HTML da UI apontando pro spec em specPath.
func DefaultSwaggerUIHTML(specPath string) string {
	return fmt.Sprintf(`<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>docwire — API Docs</title>
    <link rel="stylesheet" type="text/css" href="https://cdnjs.cloudflare.com/ajax/libs/swagger-ui/5.11.0/swagger-ui.css" />
    <style>
        /* ── reset & base ── */
        *, *:before, *:after { box-sizing: border-box; margin: 0; padding: 0; }
        html { overflow-y: scroll; }
        body {
            background: #0d1117;
            color: #e6edf3;
            font-family: 'Segoe UI', system-ui, sans-serif;
        }

        /* ── topbar ── */
        #docwire-topbar {
            display: flex;
            align-items: center;
            gap: 12px;
            padding: 0 32px;
            height: 56px;
            background: #161b22;
            border-bottom: 1px solid #21262d;
            position: sticky;
            top: 0;
            z-index: 100;
        }
        #docwire-topbar .logo-mark {
            width: 32px;
            height: 32px;
            background: linear-gradient(135deg, #39d353 0%%, #00b4d8 100%%);
            border-radius: 8px;
            display: flex;
            align-items: center;
            justify-content: center;
            font-weight: 800;
            font-size: 14px;
            color: #0d1117;
            letter-spacing: -1px;
            flex-shrink: 0;
        }
        #docwire-topbar .brand {
            font-size: 18px;
            font-weight: 700;
            letter-spacing: -0.5px;
            background: linear-gradient(90deg, #39d353, #00b4d8);
            -webkit-background-clip: text;
            -webkit-text-fill-color: transparent;
        }
        #docwire-topbar .badge {
            margin-left: auto;
            font-size: 11px;
            color: #8b949e;
            background: #21262d;
            border: 1px solid #30363d;
            border-radius: 20px;
            padding: 2px 10px;
            letter-spacing: 0.3px;
        }

        /* ── swagger-ui overrides ── */
        .swagger-ui { background: transparent; }

        /* info block */
        .swagger-ui .info { margin: 32px 0 8px; }
        .swagger-ui .info .title { color: #e6edf3 !important; font-size: 28px !important; }
        .swagger-ui .info p,
        .swagger-ui .info li { color: #8b949e !important; }
        .swagger-ui .info a { color: #39d353 !important; }

        /* scheme container / servers bar */
        .swagger-ui .scheme-container {
            background: #161b22 !important;
            border-bottom: 1px solid #21262d !important;
            box-shadow: none !important;
            padding: 12px 32px !important;
        }

        /* hide original topbar entirely */
        .swagger-ui .topbar { display: none !important; }

        /* operation blocks */
        .swagger-ui .opblock {
            background: #161b22 !important;
            border: 1px solid #21262d !important;
            border-radius: 8px !important;
            box-shadow: none !important;
            margin-bottom: 8px !important;
        }
        .swagger-ui .opblock .opblock-summary {
            border-bottom: 1px solid #21262d !important;
            border-radius: 8px 8px 0 0 !important;
        }
        .swagger-ui .opblock .opblock-summary-description { color: #8b949e !important; }
        .swagger-ui .opblock .opblock-summary-path { color: #e6edf3 !important; }

        /* GET */
        .swagger-ui .opblock.opblock-get { border-left: 3px solid #39d353 !important; }
        .swagger-ui .opblock.opblock-get .opblock-summary { background: #12201a !important; }
        .swagger-ui .opblock.opblock-get .opblock-summary-method {
            background: #39d353 !important;
            color: #0d1117 !important;
            border-radius: 4px !important;
            font-weight: 700 !important;
        }

        /* POST */
        .swagger-ui .opblock.opblock-post { border-left: 3px solid #00b4d8 !important; }
        .swagger-ui .opblock.opblock-post .opblock-summary { background: #0d1e26 !important; }
        .swagger-ui .opblock.opblock-post .opblock-summary-method {
            background: #00b4d8 !important;
            color: #0d1117 !important;
            border-radius: 4px !important;
            font-weight: 700 !important;
        }

        /* PUT / PATCH / DELETE */
        .swagger-ui .opblock.opblock-put { border-left: 3px solid #e3b341 !important; }
        .swagger-ui .opblock.opblock-put .opblock-summary { background: #1e1a0d !important; }
        .swagger-ui .opblock.opblock-put .opblock-summary-method {
            background: #e3b341 !important; color: #0d1117 !important;
            border-radius: 4px !important; font-weight: 700 !important;
        }
        .swagger-ui .opblock.opblock-delete { border-left: 3px solid #f85149 !important; }
        .swagger-ui .opblock.opblock-delete .opblock-summary { background: #1e0d0d !important; }
        .swagger-ui .opblock.opblock-delete .opblock-summary-method {
            background: #f85149 !important; color: #fff !important;
            border-radius: 4px !important; font-weight: 700 !important;
        }
        .swagger-ui .opblock.opblock-patch { border-left: 3px solid #a371f7 !important; }
        .swagger-ui .opblock.opblock-patch .opblock-summary { background: #150d1e !important; }
        .swagger-ui .opblock.opblock-patch .opblock-summary-method {
            background: #a371f7 !important; color: #0d1117 !important;
            border-radius: 4px !important; font-weight: 700 !important;
        }

        /* inner body */
        .swagger-ui .opblock-body { background: #0d1117 !important; }
        .swagger-ui .opblock-description-wrapper p,
        .swagger-ui .opblock-section-header h4,
        .swagger-ui label,
        .swagger-ui .parameter__name,
        .swagger-ui .parameter__type,
        .swagger-ui table thead tr th { color: #8b949e !important; }
        .swagger-ui .tab li,
        .swagger-ui .response-col_status { color: #e6edf3 !important; }

        /* models / schemas section */
        .swagger-ui section.models { background: #161b22 !important; border: 1px solid #21262d !important; border-radius: 8px !important; }
        .swagger-ui section.models h4 { color: #e6edf3 !important; }
        .swagger-ui .model-box { background: #0d1117 !important; }
        .swagger-ui .model { color: #8b949e !important; }
        .swagger-ui .prop-type { color: #39d353 !important; }
        .swagger-ui .prop-format { color: #00b4d8 !important; }

        /* code / response boxes */
        .swagger-ui .highlight-code,
        .swagger-ui textarea,
        .swagger-ui .microlight {
            background: #010409 !important;
            color: #e6edf3 !important;
            border: 1px solid #21262d !important;
            border-radius: 6px !important;
        }

        /* execute button */
        .swagger-ui .btn.execute {
            background: #39d353 !important;
            color: #0d1117 !important;
            border: none !important;
            border-radius: 6px !important;
            font-weight: 700 !important;
        }
        .swagger-ui .btn.execute:hover { background: #2ea043 !important; }

        /* authorize button */
        .swagger-ui .btn.authorize {
            color: #39d353 !important;
            border-color: #39d353 !important;
            border-radius: 6px !important;
        }

        /* tag group headers */
        .swagger-ui .opblock-tag {
            color: #e6edf3 !important;
            border-bottom: 1px solid #21262d !important;
            font-size: 16px !important;
        }

        /* scrollbar */
        ::-webkit-scrollbar { width: 6px; height: 6px; }
        ::-webkit-scrollbar-track { background: #0d1117; }
        ::-webkit-scrollbar-thumb { background: #30363d; border-radius: 3px; }
        ::-webkit-scrollbar-thumb:hover { background: #39d353; }
    </style>
</head>
<body>

    <div id="docwire-topbar">
        <div class="logo-mark">dw</div>
        <span class="brand">docwire</span>
        <span class="badge">API Docs</span>
    </div>

    <div id="swagger-ui"></div>

    <script src="https://cdnjs.cloudflare.com/ajax/libs/swagger-ui/5.11.0/swagger-ui-bundle.js"></script>
    <script src="https://cdnjs.cloudflare.com/ajax/libs/swagger-ui/5.11.0/swagger-ui-standalone-preset.js"></script>
    <script>
    window.onload = function() {
        window.ui = SwaggerUIBundle({
            url: "%s",
            dom_id: '#swagger-ui',
            deepLinking: true,
            presets: [
                SwaggerUIBundle.presets.apis,
                SwaggerUIStandalonePreset
            ],
            plugins: [
                SwaggerUIBundle.plugins.DownloadUrl
            ],
            layout: "StandaloneLayout"
        });
    };
    </script>
</body>
</html>`, specPath)
}
