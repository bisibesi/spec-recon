package html

// APIReportTemplate is a Swagger/Redoc-style HTML template for API documentation
const APIReportTemplate = `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>API Specification - {{.AnalysisDate}}</title>
    <style>
        * {
            margin: 0;
            padding: 0;
            box-sizing: border-box;
        }

        body {
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, 'Helvetica Neue', Arial, sans-serif;
            background: #f5f7fa;
            color: #2c3e50;
            line-height: 1.6;
        }

        .container {
            max-width: 1200px;
            margin: 0 auto;
            padding: 20px;
        }

        header {
            background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
            color: white;
            padding: 40px 20px;
            margin-bottom: 30px;
            border-radius: 8px;
            box-shadow: 0 4px 6px rgba(0, 0, 0, 0.1);
        }

        header h1 {
            font-size: 2.5em;
            margin-bottom: 10px;
        }

        header p {
            font-size: 1.1em;
            opacity: 0.9;
        }

        .summary {
            background: white;
            padding: 20px;
            border-radius: 8px;
            margin-bottom: 30px;
            box-shadow: 0 2px 4px rgba(0, 0, 0, 0.05);
        }

        .summary h2 {
            color: #667eea;
            margin-bottom: 15px;
            font-size: 1.5em;
        }

        .stats {
            display: grid;
            grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
            gap: 15px;
            margin-top: 15px;
        }

        .stat-card {
            background: #f8f9fa;
            padding: 15px;
            border-radius: 6px;
            border-left: 4px solid #667eea;
        }

        .stat-card .label {
            font-size: 0.9em;
            color: #6c757d;
            margin-bottom: 5px;
        }

        .stat-card .value {
            font-size: 1.8em;
            font-weight: bold;
            color: #2c3e50;
        }

        .endpoint {
            background: white;
            margin-bottom: 20px;
            border-radius: 8px;
            overflow: hidden;
            box-shadow: 0 2px 4px rgba(0, 0, 0, 0.05);
            transition: box-shadow 0.3s ease;
        }

        .endpoint:hover {
            box-shadow: 0 4px 12px rgba(0, 0, 0, 0.1);
        }

        .endpoint-header {
            padding: 20px;
            background: #f8f9fa;
            border-bottom: 1px solid #e9ecef;
            cursor: pointer;
        }

        .endpoint-title {
            display: flex;
            align-items: center;
            gap: 15px;
            margin-bottom: 10px;
        }

        .method-badge {
            display: inline-block;
            padding: 6px 12px;
            border-radius: 4px;
            font-weight: bold;
            font-size: 0.85em;
            text-transform: uppercase;
            letter-spacing: 0.5px;
        }

        .method-get { background: #61affe; color: white; }
        .method-post { background: #49cc90; color: white; }
        .method-put { background: #fca130; color: white; }
        .method-delete { background: #f93e3e; color: white; }
        .method-patch { background: #50e3c2; color: white; }
        .method-default { background: #6c757d; color: white; }

        .endpoint-path {
            font-size: 1.3em;
            font-weight: 600;
            color: #2c3e50;
            font-family: 'Courier New', monospace;
        }

        .endpoint-meta {
            font-size: 0.9em;
            color: #6c757d;
            margin-top: 5px;
        }

        .endpoint-summary {
            margin-top: 10px;
            color: #495057;
        }

        .endpoint-body {
            padding: 20px;
        }

        .section-title {
            font-size: 1.1em;
            font-weight: 600;
            color: #495057;
            margin-bottom: 15px;
            padding-bottom: 8px;
            border-bottom: 2px solid #e9ecef;
        }

        table {
            width: 100%;
            border-collapse: collapse;
            margin-bottom: 20px;
        }

        th {
            background: #f8f9fa;
            padding: 12px;
            text-align: left;
            font-weight: 600;
            color: #495057;
            border-bottom: 2px solid #dee2e6;
        }

        td {
            padding: 12px;
            border-bottom: 1px solid #e9ecef;
        }

        tr:hover {
            background: #f8f9fa;
        }

        .param-name {
            font-family: 'Courier New', monospace;
            color: #667eea;
            font-weight: 600;
        }

        .param-type {
            font-family: 'Courier New', monospace;
            color: #e83e8c;
        }

        .param-in {
            display: inline-block;
            padding: 2px 8px;
            background: #e7f3ff;
            color: #0066cc;
            border-radius: 3px;
            font-size: 0.85em;
            font-weight: 500;
        }

        .required-badge {
            display: inline-block;
            padding: 2px 6px;
            background: #f93e3e;
            color: white;
            border-radius: 3px;
            font-size: 0.75em;
            font-weight: bold;
        }

        .optional-badge {
            display: inline-block;
            padding: 2px 6px;
            background: #6c757d;
            color: white;
            border-radius: 3px;
            font-size: 0.75em;
        }

        .response-success {
            color: #49cc90;
            font-weight: 600;
        }

        footer {
            text-align: center;
            padding: 30px 20px;
            color: #6c757d;
            margin-top: 40px;
        }

        .no-endpoints {
            text-align: center;
            padding: 60px 20px;
            color: #6c757d;
        }
        
        /* Nested field styling */
        .nested-param {
            background: #fcfcfc;
        }
        
        .nested-param .param-name {
            color: #6c757d;
            font-size: 0.95em;
        }
        
        .nested-param .param-type {
            color: #6c757d;
            font-size: 0.95em;
        }
    </style>
</head>
<body>
    <div class="container">
        <header>
            <h1>📘 API Specification</h1>
            <p>Generated on {{.AnalysisDate}}</p>
        </header>

        <div class="summary">
            <h2>Overview</h2>
            <div class="stats">
                <div class="stat-card">
                    <div class="label">Total Endpoints</div>
                    <div class="value">{{.TotalEndpoints}}</div>
                </div>
                <div class="stat-card">
                    <div class="label">Controllers</div>
                    <div class="value">{{.TotalControllers}}</div>
                </div>
            </div>
        </div>

        {{if .Endpoints}}
            {{range .Endpoints}}
            <div class="endpoint">
                <div class="endpoint-header">
                    <div class="endpoint-title">
                        <span class="method-badge {{methodColor .Method}}">{{methodBadge .Method}}</span>
                        <span class="endpoint-path">{{.Path}}</span>
                    </div>
                    <div class="endpoint-meta">
                        Controller: <strong>{{.ControllerName}}</strong> · Method: <strong>{{.MethodName}}</strong>
                    </div>
                    {{if .Summary}}
                    <div class="endpoint-summary">{{.Summary}}</div>
                    {{end}}
                </div>

                <div class="endpoint-body">
                    {{if .Params}}
                    <div class="section-title">Request Parameters</div>
                    <table>
                        <thead>
                            <tr>
                                <th>Name</th>
                                <th>Type</th>
                                <th>In</th>
                                <th>Required</th>
                                <th>Description</th>
                            </tr>
                        </thead>
                        <tbody>
                            {{range .Params}}
                            <tr>
                                <td class="param-name">{{.Name}}</td>
                                <td class="param-type">{{.Type}}</td>
                                <td><span class="param-in">{{.In}}</span></td>
                                <td>
                                    {{if .Required}}
                                    <span class="required-badge">REQUIRED</span>
                                    {{else}}
                                    <span class="optional-badge">Optional</span>
                                    {{end}}
                                </td>
                                <td>{{.Description}}</td>
                            </tr>
                            {{if .Fields}}
                                {{range .Fields}}
                                <tr class="nested-param">
                                    <td class="param-name" style="padding-left: {{mul .Depth 20}}px">
                                        {{if gt .Depth 0}}└ {{end}}{{.Name}}
                                    </td>
                                    <td class="param-type">{{.Type}}</td>
                                    <td>-</td>
                                    <td>-</td>
                                    <td>{{.Description}}</td>
                                </tr>
                                {{end}}
                            {{end}}
                            {{end}}
                        </tbody>
                    </table>
                    {{end}}

                    <div class="section-title">Response</div>
                    <table>
                        <thead>
                            <tr>
                                <th>Status Code</th>
                                <th>Type</th>
                                <th>Description</th>
                            </tr>
                        </thead>
                        <tbody>
                            <tr>
                                <td class="response-success">{{.Response.StatusCode}}</td>
                                <td class="param-type">{{.Response.Type}}</td>
                                <td>{{.Response.Description}}</td>
                            </tr>
                        </tbody>
                    </table>
                    
                    {{if .Response.Fields}}
                    <div class="section-title">Response Fields</div>
                    <table>
                        <thead>
                            <tr>
                                <th>Field Name</th>
                                <th>Type</th>
                                <th>Description</th>
                            </tr>
                        </thead>
                        <tbody>
                            {{range .Response.Fields}}
                            <tr class="nested-param">
                                <td class="param-name" style="padding-left: {{mul (sub .Depth 1) 20}}px">
                                    {{if gt (sub .Depth 1) 0}}└ {{end}}{{.Name}}
                                </td>
                                <td class="param-type">{{.Type}}</td>
                                <td>{{.Description}}</td>
                            </tr>
                            {{end}}
                        </tbody>
                    </table>
                    {{end}}
                </div>
            </div>
            {{end}}
        {{else}}
            <div class="no-endpoints">
                <h3>No API endpoints found</h3>
                <p>Make sure your controllers have proper @RequestMapping annotations.</p>
            </div>
        {{end}}

        <footer>
            <p>Generated by <strong>Spec Recon</strong> v1.0.0</p>
            <p>Static Analysis for Legacy Spring Projects</p>
        </footer>
    </div>
</body>
</html>
`
