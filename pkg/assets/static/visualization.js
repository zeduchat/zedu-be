// Standalone Visualization Module
// This file contains all visualization functionality separated from the main script

function createStandaloneVisualization(workflowData) {
  // Create the complete HTML structure with embedded CSS and dependencies
  const htmlContent = `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Workflow Visualization</title>
    <script type="text/javascript" src="https://unpkg.com/vis-network/standalone/umd/vis-network.min.js"></script>
    <style>
        :root {
            --bg-primary: #ffffff;
            --bg-secondary: #f8f9fa;
            --bg-tertiary: #e9ecef;
            --text-primary: #212529;
            --text-secondary: #6c757d;
            --text-muted: #868e96;
            --border-color: #dee2e6;
            --accent-primary: #2563eb;
            --accent-success: #10b981;
            --shadow-sm: 0 1px 2px 0 rgba(0, 0, 0, 0.05);
            --shadow-md: 0 4px 6px -1px rgba(0, 0, 0, 0.1), 0 2px 4px -1px rgba(0, 0, 0, 0.06);
            --border-radius: 8px;
        }

        * {
            margin: 0;
            padding: 0;
            box-sizing: border-box;
        }

        body {
            font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif;
            background: var(--bg-primary);
            color: var(--text-primary);
            height: 100vh;
            overflow: hidden;
        }

        .visualization-container {
            position: relative;
            width: 100vw;
            height: 100vh;
            display: flex;
            flex-direction: column;
        }

        .visualization-header {
            background: var(--bg-secondary);
            border-bottom: 1px solid var(--border-color);
            padding: 12px 20px;
            display: flex;
            justify-content: space-between;
            align-items: center;
            z-index: 1000;
        }

        .visualization-title {
            font-size: 18px;
            font-weight: 600;
            color: var(--text-primary);
        }

        .visualization-controls {
            display: flex;
            gap: 8px;
            align-items: center;
        }

        .viz-control-btn {
            background: var(--bg-primary);
            border: 1px solid var(--border-color);
            border-radius: var(--border-radius);
            padding: 8px 12px;
            cursor: pointer;
            font-size: 14px;
            transition: all 0.2s ease;
            display: flex;
            align-items: center;
            gap: 4px;
        }

        .viz-control-btn:hover {
            background: var(--bg-tertiary);
            border-color: var(--accent-primary);
        }

        .workflow-network {
            flex: 1;
            background: linear-gradient(135deg, #f8f9fa 0%, #e9ecef 100%);
            border: none;
            position: relative;
        }

        .visualization-stats {
            position: absolute;
            bottom: 10px;
            left: 10px;
            z-index: 1000;
            background: rgba(0,0,0,0.7);
            color: white;
            padding: 8px 12px;
            border-radius: 4px;
            font-size: 12px;
            font-family: "SF Mono", Monaco, "Cascadia Code", "Roboto Mono", Consolas, "Courier New", monospace;
        }

        .visualization-info {
            position: absolute;
            top: 10px;
            left: 10px;
            z-index: 1000;
            background: rgba(255,255,255,0.9);
            padding: 12px;
            border-radius: var(--border-radius);
            box-shadow: var(--shadow-md);
            max-width: 300px;
            font-size: 12px;
            line-height: 1.4;
        }

        .vis-tooltip {
            background: rgba(0, 0, 0, 0.9) !important;
            color: white !important;
            border: none !important;
            border-radius: 6px !important;
            padding: 12px !important;
            font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif !important;
            font-size: 12px !important;
            line-height: 1.4 !important;
            box-shadow: 0 4px 12px rgba(0, 0, 0, 0.3) !important;
        }

        @media (prefers-color-scheme: dark) {
            :root {
                --bg-primary: #1a1a1a;
                --bg-secondary: #2d2d2d;
                --bg-tertiary: #404040;
                --text-primary: #ffffff;
                --text-secondary: #e0e0e0;
                --text-muted: #a0a0a0;
                --border-color: #404040;
            }

            .workflow-network {
                background: linear-gradient(135deg, #1a1a1a 0%, #2d2d2d 100%) !important;
            }

            .visualization-info {
                background: rgba(45,45,45,0.9);
                color: var(--text-primary);
            }
        }
    </style>
</head>
<body>
    <div class="visualization-container">
        <div class="visualization-header">
            <h1 class="visualization-title">Workflow Visualization</h1>
            <div class="visualization-controls" id="visualizationControls">
                <button id="darkModeToggle" class="viz-control-btn" title="Toggle Dark Mode">🌙</button>
                <button id="fitViewBtn" class="viz-control-btn" title="Fit to View">📐</button>
                <button id="resetLayoutBtn" class="viz-control-btn" title="Reset Layout">🔄</button>
                <button id="togglePhysicsBtn" class="viz-control-btn" title="Toggle Physics">⚡</button>
                <button id="exportImageBtn" class="viz-control-btn" title="Export as Image">📷</button>
            </div>
        </div>
        
        <div id="workflowNetwork" class="workflow-network"></div>
        
        <div class="visualization-info">
            <strong>Instructions:</strong> Drag nodes to rearrange, zoom with mouse wheel, click and drag to pan. Hover over nodes to see full details.
        </div>
        
        <div class="visualization-stats">
            <span id="nodeCount">Nodes: 0</span> | 
            <span id="edgeCount">Connections: 0</span>
        </div>
    </div>

    <script>
        // Embedded workflow data
        const workflowData = ${JSON.stringify(workflowData, null, 2)};
        
        // Extract skill from task name
        function extractSkillFromTask(taskName) {
            if (!taskName) return 'Unknown Task';
            
            const skillPatterns = [
                /skill:\\s*([^,\\n]+)/i,
                /using\\s+([a-zA-Z_][a-zA-Z0-9_]*)/i,
                /with\\s+([a-zA-Z_][a-zA-Z0-9_]*)/i,
                /([a-zA-Z_][a-zA-Z0-9_]*)\\s*\\(/,
                /\\b([a-zA-Z_][a-zA-Z0-9_]*)\\s*$/
            ];
            
            for (const pattern of skillPatterns) {
                const match = taskName.match(pattern);
                if (match && match[1]) {
                    return match[1].trim();
                }
            }
            
            const words = taskName.split(/\\s+/);
            return words.length > 2 ? words.slice(0, 2).join(' ') + '...' : taskName;
        }

        // Get node color based on type
        function getNodeColor(nodeType, isDarkMode) {
            const colorMap = {
                "n8n-nodes-base.start": isDarkMode
                    ? { background: "#4caf50", border: "#388e3c" }
                    : { background: "#66bb6a", border: "#4caf50" },
                "n8n-nodes-base.googleSheets": isDarkMode
                    ? { background: "#2196f3", border: "#1976d2" }
                    : { background: "#42a5f5", border: "#2196f3" },
                "n8n-nodes-base.gmail": isDarkMode
                    ? { background: "#f44336", border: "#d32f2f" }
                    : { background: "#ef5350", border: "#f44336" },
                "n8n-nodes-base.code": isDarkMode
                    ? { background: "#ff9800", border: "#f57c00" }
                    : { background: "#ffa726", border: "#ff9800" },
                "n8n-nodes-base.set": isDarkMode
                    ? { background: "#9c27b0", border: "#7b1fa2" }
                    : { background: "#ab47bc", border: "#9c27b0" },
            };

            return colorMap[nodeType] || (isDarkMode 
                ? { background: "#607d8b", border: "#455a64" } 
                : { background: "#78909c", border: "#607d8b" });
        }

        // Create visualization
        function createVisualization() {
            const container = document.getElementById('workflowNetwork');
            const nodes = [];
            const edges = [];
            const isDarkMode = window.matchMedia('(prefers-color-scheme: dark)').matches;

            // Create nodes with node_name field support
            workflowData.nodes.forEach((node, index) => {
                const displayName = node.node_name || extractSkillFromTask(node.name);
                const realName = node.name;

                let x, y;
                if (node.position && node.position.length >= 2) {
                    x = node.position[0];
                    y = node.position[1];
                } else {
                    x = index * 280 + 150;
                    y = 200 + (index % 3) * 120;
                }

                const nodeColor = getNodeColor(node.type, isDarkMode);

                nodes.push({
                    id: node.id,
                    label: displayName, // Display node_name instead of extracted skill
                    shape: "box",
                    color: nodeColor,
                    font: {
                        color: isDarkMode ? "#ffffff" : "#333333",
                        size: 13,
                        face: "Inter, -apple-system, sans-serif",
                    },
                    title: \`<div style="max-width: 350px; white-space: normal; line-height: 1.5; font-family: Inter, sans-serif;">
                        <strong style="color: \${nodeColor.background};">Real Name:</strong><br/>
                        <span style="font-size: 12px;">\${realName}</span><br/><br/>
                        <strong>Display Name:</strong> \${displayName}<br/>
                        <strong>Type:</strong> \${node.type}<br/>
                        <strong>ID:</strong> \${node.id}
                    </div>\`,
                    x: x,
                    y: y,
                    fixed: { x: false, y: false },
                    borderWidth: 2,
                    borderWidthSelected: 3,
                    margin: 12,
                });
            });

            // Create edges
            let edgeCount = 0;
            if (workflowData.connections) {
                Object.keys(workflowData.connections).forEach((sourceNodeName) => {
                    const connections = workflowData.connections[sourceNodeName];
                    const sourceNode = workflowData.nodes.find((n) => n.name === sourceNodeName);

                    if (!sourceNode) return;

                    if (connections.main && Array.isArray(connections.main)) {
                        connections.main.forEach((outputArray, outputIndex) => {
                            if (Array.isArray(outputArray)) {
                                outputArray.forEach((connection, connIndex) => {
                                    const targetNode = workflowData.nodes.find((n) => n.name === connection.node);
                                    if (targetNode) {
                                        edges.push({
                                            id: \`\${sourceNode.id}-\${targetNode.id}-\${outputIndex}-\${connIndex}\`,
                                            from: sourceNode.id,
                                            to: targetNode.id,
                                            arrows: { to: { enabled: true, scaleFactor: 1.2 } },
                                            color: {
                                                color: isDarkMode ? "#64b5f6" : "#1976d2",
                                                highlight: isDarkMode ? "#90caf9" : "#1565c0",
                                            },
                                            width: 2,
                                            smooth: { type: "cubicBezier", forceDirection: "horizontal", roundness: 0.4 },
                                            label: outputIndex > 0 ? \`out\${outputIndex}\` : "",
                                            font: { size: 10, color: isDarkMode ? "#ffffff" : "#666666" },
                                        });
                                        edgeCount++;
                                    }
                                });
                            }
                        });
                    }
                });
            }

            // Update stats
            document.getElementById('nodeCount').textContent = \`Nodes: \${nodes.length}\`;
            document.getElementById('edgeCount').textContent = \`Connections: \${edgeCount}\`;

            const options = {
                layout: {
                    hierarchical: { enabled: false },
                },
                physics: {
                    enabled: true,
                    stabilization: { iterations: 100 },
                    barnesHut: {
                        gravitationalConstant: -8000,
                        centralGravity: 0.3,
                        springLength: 200,
                        springConstant: 0.04,
                        damping: 0.09,
                    },
                },
                interaction: {
                    dragNodes: true,
                    dragView: true,
                    zoomView: true,
                    hover: true,
                    selectConnectedEdges: false,
                    tooltipDelay: 200,
                },
                nodes: {
                    margin: 15,
                    font: { size: 13 },
                    widthConstraint: { minimum: 140, maximum: 220 },
                    heightConstraint: { minimum: 40 },
                    shadow: {
                        enabled: true,
                        color: isDarkMode ? "rgba(0,0,0,0.5)" : "rgba(0,0,0,0.2)",
                        size: isDarkMode ? 10 : 8,
                        x: 2,
                        y: 2,
                    },
                },
                edges: {
                    smooth: {
                        type: "cubicBezier",
                        forceDirection: "horizontal",
                        roundness: 0.4,
                    },
                    width: 2,
                    shadow: {
                        enabled: true,
                        color: "rgba(0,0,0,0.2)",
                        size: 5,
                        x: 1,
                        y: 1,
                    },
                },
            };

            const networkData = {
                nodes: new vis.DataSet(nodes),
                edges: new vis.DataSet(edges),
            };

            const network = new vis.Network(container, networkData, options);
            
            // Setup controls
            setupControls(network, networkData, isDarkMode);

            // Auto-fit on load
            setTimeout(() => {
                network.fit({ animation: { duration: 1000, easingFunction: "easeInOutQuad" } });
            }, 500);
        }

        // Setup control buttons
        function setupControls(network, networkData, initialDarkMode) {
            let isDarkMode = initialDarkMode;
            let physicsEnabled = true;

            // Dark mode toggle
            const darkModeBtn = document.getElementById('darkModeToggle');
            darkModeBtn.textContent = isDarkMode ? "☀️" : "🌙";
            darkModeBtn.onclick = () => {
                isDarkMode = !isDarkMode;
                darkModeBtn.textContent = isDarkMode ? "☀️" : "🌙";
                
                const container = document.getElementById('workflowNetwork');
                if (isDarkMode) {
                    container.style.background = "linear-gradient(135deg, #1a1a1a 0%, #2d2d2d 100%)";
                } else {
                    container.style.background = "linear-gradient(135deg, #f8f9fa 0%, #e9ecef 100%)";
                }

                // Update node colors
                const nodes = networkData.nodes.get();
                nodes.forEach((node) => {
                    const newColor = getNodeColor(node.type || "default", isDarkMode);
                    networkData.nodes.update({
                        id: node.id,
                        color: newColor,
                        font: { ...node.font, color: isDarkMode ? "#ffffff" : "#333333" },
                    });
                });
            };

            // Fit view
            document.getElementById('fitViewBtn').onclick = () => {
                network.fit({ animation: { duration: 800, easingFunction: "easeInOutQuad" } });
            };

            // Reset layout
            document.getElementById('resetLayoutBtn').onclick = () => {
                network.setData(networkData);
                setTimeout(() => {
                    network.fit({ animation: { duration: 1000, easingFunction: "easeInOutQuad" } });
                }, 100);
            };

            // Toggle physics
            const physicsBtn = document.getElementById('togglePhysicsBtn');
            physicsBtn.onclick = () => {
                physicsEnabled = !physicsEnabled;
                network.setOptions({ physics: { enabled: physicsEnabled } });
                physicsBtn.style.opacity = physicsEnabled ? "1" : "0.5";
                physicsBtn.title = physicsEnabled ? "Disable Physics" : "Enable Physics";
            };

            // Export image
            document.getElementById('exportImageBtn').onclick = () => {
                const canvas = document.querySelector('#workflowNetwork canvas');
                if (canvas) {
                    const link = document.createElement('a');
                    link.download = 'workflow-visualization.png';
                    link.href = canvas.toDataURL();
                    link.click();
                }
            };
        }

        // Initialize visualization when page loads
        document.addEventListener('DOMContentLoaded', createVisualization);
    </script>
</body>
</html>`

  return htmlContent
}

// Export function for use in main script
if (typeof module !== "undefined" && module.exports) {
  module.exports = { createStandaloneVisualization }
}
