// Global variables
const tasks = []
let taskMarkdown = "" // Store single markdown string instead of array
const agentSkills = []
const globalSkills = []
let selectedSteps = []
let availablePrompts = []
let lastTranslationResponse = null
let selectedVersionId = null
let currentPromptVersions = null
const BASE_URL = "/api/v1"

// Import vis library
const vis = window.vis || {}
const marked = window.marked || {} // Import marked.js library

// Tab functionality
document.addEventListener("DOMContentLoaded", () => {
  initializeTabs()
  loadAvailablePrompts()
  updateAllDisplays()
})

function initializeTabs() {
  const tabBtns = document.querySelectorAll(".tab-btn")
  const tabContents = document.querySelectorAll(".tab-content")

  tabBtns.forEach((btn) => {
    btn.addEventListener("click", () => {
      const targetTab = btn.getAttribute("data-tab")

      // Remove active class from all tabs and contents
      tabBtns.forEach((b) => b.classList.remove("active"))
      tabContents.forEach((c) => c.classList.remove("active"))

      // Add active class to clicked tab and corresponding content
      btn.classList.add("active")
      document.getElementById(`${targetTab}-tab`).classList.add("active")

      // Update summaries when switching to execute tab
      if (targetTab === "execute") {
        updateExecutionSummary()
      }
    })
  })
}

function addTaskMarkdown() {
  const textarea = document.getElementById("taskMarkdown")
  const markdown = textarea.value.trim()

  if (!markdown) {
    alert("Please enter task markdown")
    return
  }

  taskMarkdown = markdown
  updateTaskDisplay()

  // Hide input section after adding task
  document.getElementById("taskInputSection").style.display = "none"
}

function clearTasks() {
  taskMarkdown = ""
  updateTaskDisplay()
  // Show input section again when clearing
  document.getElementById("taskInputSection").style.display = "block"
}

function updateTaskDisplay() {
  const displayArea = document.getElementById("taskDisplayArea")
  const emptyState = document.getElementById("emptyTaskState")
  const markdownDisplay = document.getElementById("taskMarkdownDisplay")

  if (!taskMarkdown) {
    emptyState.style.display = "flex"
    markdownDisplay.style.display = "none"
  } else {
    emptyState.style.display = "none"
    markdownDisplay.style.display = "block"

    const parsedMarkdown = marked.parse(taskMarkdown)
    document.getElementById("parsedMarkdown").innerHTML = parsedMarkdown
  }
}

function addSkillsBulk(type) {
  const textarea = document.getElementById(`${type}SkillsBulk`)
  const skills = textarea.value
    .split("\n")
    .map((s) => s.trim())
    .filter((s) => s)

  if (skills.length > 0) {
    if (type === "agent") {
      agentSkills.push(...skills)
    } else {
      globalSkills.push(...skills)
    }

    textarea.value = ""
    updateSkillsDisplay(type)
  }
}

function removeSkill(type, index) {
  if (type === "agent") {
    agentSkills.splice(index, 1)
  } else {
    globalSkills.splice(index, 1)
  }
  updateSkillsDisplay(type)
}

function updateSkillsDisplay(type) {
  const displayArea = document.getElementById(`${type}SkillsDisplay`)
  const skillsArray = type === "agent" ? agentSkills : globalSkills
  const countElement = document.getElementById(`${type}SkillsCount`)

  // Update count
  countElement.textContent = `${skillsArray.length} skills`

  if (skillsArray.length === 0) {
    displayArea.innerHTML = `
            <div class="empty-state">
                <div class="empty-icon">${type === "agent" ? "🤖" : "🌍"}</div>
                <p>No ${type} skills configured</p>
            </div>
        `
  } else {
    displayArea.innerHTML = `<div class="skills-list" id="${type}SkillsList"></div>`
    const skillsList = document.getElementById(`${type}SkillsList`)

    skillsArray.forEach((skill, index) => {
      const skillTag = document.createElement("div")
      skillTag.className = "skill-tag"
      skillTag.innerHTML = `
                ${escapeHtml(skill)}
                <span class="remove-skill" onclick="removeSkill('${type}', ${index})">×</span>
            `
      skillsList.appendChild(skillTag)
    })
  }
}

function updateAllDisplays() {
  updateTaskDisplay()
  updateSkillsDisplay("agent")
  updateSkillsDisplay("global")
  updatePipelineStepsDisplay()
}

async function loadAvailablePrompts() {
  const loadingIndicator = document.getElementById("pipelineLoading")
  loadingIndicator.style.display = "flex"

  try {
    const response = await fetch(`${BASE_URL}/prompts/steps`)
    const data = await response.json()

    if (data.status === "success") {
      // Extract unique prompt names
      const uniqueNames = [...new Set(data.data.map((step) => step.name))]
      availablePrompts = uniqueNames
      displayAvailablePrompts(availablePrompts)
    } else {
      console.error("Failed to load prompts:", data.message)
      showPromptsError(`Failed to load prompts: ${data.message}`)
    }
  } catch (error) {
    console.error("Error loading prompts:", error)
    showPromptsError("Error loading prompts. Please check your connection and try again.")
  } finally {
    loadingIndicator.style.display = "none"
  }
}

function displayAvailablePrompts(availablePrompts) {
  const container = document.getElementById("promptsContainer")

  if (!availablePrompts || availablePrompts.length === 0) {
    container.innerHTML = `
            <div style="padding: 20px; text-align: center; color: var(--text-secondary);">
                No prompts available
            </div>
        `
    return
  }

  container.innerHTML = ""

  availablePrompts.forEach((promptName) => {
    const promptElement = document.createElement("div")
    promptElement.className = "prompt-item"

    promptElement.innerHTML = `
            <div class="prompt-name">${escapeHtml(promptName)}</div>
            <button class="view-versions-btn" onclick="showPromptVersions('${escapeHtml(promptName)}')">
                View Versions
            </button>
        `

    container.appendChild(promptElement)
  })
}

function updatePipelineStepsDisplay() {
  const stepsArea = document.getElementById("pipelineStepsArea")
  const stepsCount = document.getElementById("stepsCount")

  stepsCount.textContent = `${selectedSteps.length} steps`

  if (selectedSteps.length === 0) {
    stepsArea.innerHTML = `
            <div class="empty-state">
                <div class="empty-icon">⚙️</div>
                <h4>No steps selected</h4>
                <p>Click on prompts from the left to add them as pipeline steps</p>
            </div>
        `
  } else {
    stepsArea.innerHTML = `
      <div class="drag-instruction">
        <span class="drag-icon">↕️</span>
        <span>Drag steps to reorder</span>
      </div>
    `

    selectedSteps.forEach((step, index) => {
      const stepElement = document.createElement("div")
      stepElement.className = "pipeline-step"
      stepElement.draggable = true
      stepElement.dataset.index = index

      stepElement.innerHTML = `
                <div class="step-info">
                    <div class="step-number">${index + 1}</div>
                    <div class="step-details">
                        <div class="step-name">${escapeHtml(step.name)}</div>
                        <div class="step-version">Version ${step.version}</div>
                    </div>
                    <div class="drag-handle">⋮⋮</div>
                </div>
                <button class="remove-step-btn" onclick="removeStep(${index})">Remove</button>
            `

      stepElement.addEventListener("dragstart", handleDragStart)
      stepElement.addEventListener("dragover", handleDragOver)
      stepElement.addEventListener("drop", handleDrop)
      stepElement.addEventListener("dragend", handleDragEnd)

      stepsArea.appendChild(stepElement)
    })
  }
}

let draggedElement = null

function handleDragStart(e) {
  draggedElement = this
  this.classList.add("dragging")
  e.dataTransfer.effectAllowed = "move"
  e.dataTransfer.setData("text/html", this.outerHTML)
}

function handleDragOver(e) {
  if (e.preventDefault) {
    e.preventDefault()
  }
  e.dataTransfer.dropEffect = "move"
  return false
}

function handleDrop(e) {
  if (e.stopPropagation) {
    e.stopPropagation()
  }

  if (draggedElement !== this) {
    const draggedIndex = Number.parseInt(draggedElement.dataset.index)
    const targetIndex = Number.parseInt(this.dataset.index)

    const draggedStep = selectedSteps[draggedIndex]
    selectedSteps.splice(draggedIndex, 1)
    selectedSteps.splice(targetIndex, 0, draggedStep)

    updatePipelineStepsDisplay()
  }
  return false
}

function handleDragEnd(e) {
  this.classList.remove("dragging")
  draggedElement = null
}

function removeStep(index) {
  selectedSteps.splice(index, 1)
  updatePipelineStepsDisplay()
}

function showPromptsError(message) {
  const container = document.getElementById("promptsContainer")
  container.innerHTML = `
        <div style="padding: 20px; text-align: center; color: var(--accent-danger); background: rgba(239, 68, 68, 0.1); border-radius: 4px; border: 1px solid var(--accent-danger);">
            ${escapeHtml(message)}
        </div>
    `
}

async function refreshPipelineSteps() {
  const refreshBtn = document.querySelector(".refresh-btn")
  const refreshIcon = document.querySelector(".refresh-icon")
  const loadingIndicator = document.getElementById("pipelineLoading")

  refreshBtn.disabled = true
  refreshIcon.style.animation = "spin 1s linear infinite"
  loadingIndicator.style.display = "flex"

  selectedSteps = []

  await loadAvailablePrompts()

  refreshBtn.disabled = false
  refreshIcon.style.animation = "none"
  loadingIndicator.style.display = "none"

  updatePipelineStepsDisplay()
}

function showAddPromptModal() {
  document.getElementById("addPromptModal").style.display = "flex"
  setPromptModalMode("new")
}

function hideAddPromptModal() {
  document.getElementById("addPromptModal").style.display = "none"
  document.getElementById("promptName").value = ""
  document.getElementById("promptContent").value = ""
}

function setPromptModalMode(mode) {
  const modalTitle = document.querySelector("#addPromptModal .modal-header h3")
  const promptNameGroup = document.getElementById("promptNameGroup")
  const promptSelectGroup = document.getElementById("promptSelectGroup")
  const newPromptBtn = document.getElementById("newPromptBtn")
  const updatePromptBtn = document.getElementById("updatePromptBtn")

  if (mode === "new") {
    modalTitle.textContent = "Add New Prompt"
    promptNameGroup.style.display = "block"
    promptSelectGroup.style.display = "none"
    newPromptBtn.classList.add("active")
    updatePromptBtn.classList.remove("active")
  } else {
    modalTitle.textContent = "Update Existing Prompt"
    promptNameGroup.style.display = "none"
    promptSelectGroup.style.display = "block"
    newPromptBtn.classList.remove("active")
    updatePromptBtn.classList.add("active")

    populatePromptSelect()
  }
}

function populatePromptSelect() {
  const select = document.getElementById("promptSelect")
  select.innerHTML = '<option value="">Select a prompt to update...</option>'

  availablePrompts.forEach((promptName) => {
    const option = document.createElement("option")
    option.value = promptName
    option.textContent = promptName
    select.appendChild(option)
  })
}

async function saveNewPrompt() {
  const mode = document.getElementById("newPromptBtn").classList.contains("active") ? "new" : "update"
  let name, template

  if (mode === "new") {
    name = document.getElementById("promptName").value.trim()
    template = document.getElementById("promptContent").value.trim()
  } else {
    name = document.getElementById("promptSelect").value
    template = document.getElementById("promptContent").value.trim()
  }

  if (!name || !template) {
    alert("Please fill in all required fields.")
    return
  }

  try {
    const response = await fetch(`${BASE_URL}/prompts`, {
      method: "POST",
      headers: {
        "Content-Type": "application/json",
      },
      body: JSON.stringify({ name, template }),
    })

    const result = await response.json()

    if (response.ok && result.status === "success") {
      hideAddPromptModal()
      alert(mode === "new" ? "Prompt saved successfully!" : "Prompt updated successfully!")
      await loadAvailablePrompts()
    } else {
      alert(`Failed to ${mode === "new" ? "save" : "update"} prompt: ${result.message || "Unknown error"}`)
    }
  } catch (error) {
    console.error("Error saving prompt:", error)
    alert("Error saving prompt. Please check your connection and try again.")
  }
}

function updateExecutionSummary() {
  document.getElementById("taskSummary").textContent = taskMarkdown ? "1 task list" : "0 tasks"
  document.getElementById("agentSkillsSummary").textContent = `${agentSkills.length} skills`
  document.getElementById("globalSkillsSummary").textContent = `${globalSkills.length} skills`
  document.getElementById("stepsSummary").textContent = `${selectedSteps.length} steps`
}

function validateTranslationData() {
  const errors = []
  if (!taskMarkdown.trim()) {
    errors.push("Please add task markdown.")
  }
  if (selectedSteps.length === 0) {
    errors.push("Please select at least one pipeline step.")
  }
  return errors
}

function showValidationErrors(errors) {
  const errorContainer = document.getElementById("validationErrors")
  errorContainer.innerHTML = "<ul>" + errors.map((e) => `<li>${e}</li>`).join("") + "</ul>"
  errorContainer.style.display = "block"
}

function hideValidationErrors() {
  const errorContainer = document.getElementById("validationErrors")
  errorContainer.style.display = "none"
}

async function performTranslation() {
  hideValidationErrors()
  const validationErrors = validateTranslationData()
  if (validationErrors.length > 0) {
    showValidationErrors(validationErrors)
    return
  }

  const loadingOverlay = document.getElementById("loadingOverlay")
  const allInputsAndButtons = document.querySelectorAll("input, button, textarea")

  loadingOverlay.style.display = "flex"
  allInputsAndButtons.forEach((el) => (el.disabled = true))

  try {
    const payload = {
      task_list: taskMarkdown,
      agent_skills: agentSkills,
      global_skills: globalSkills,
      steps: selectedSteps.map((step) => ({
        name: step.name,
        version: step.version,
      })),
    }

    console.log("Sending translation request:", payload)

    const response = await fetch(`${BASE_URL}/translator`, {
      method: "POST",
      headers: {
        "Content-Type": "application/json",
      },
      body: JSON.stringify(payload),
    })

    const result = await response.json()
    lastTranslationResponse = result

    console.log("Translation response:", result)

    if (response.ok && result.status === "success") {
      renderResults(result.data)
      document.querySelector('.tab-btn[data-tab="results"]').click()
    } else {
      const error = result.message || "An unknown error occurred."
      showValidationErrors(["API Error: " + error])
      document.querySelector('.tab-btn[data-tab="execute"]').click()
    }
  } catch (error) {
    console.error("Translation Error:", error)
    showValidationErrors(["A network or system error occurred. Please check the console for more details."])
    document.querySelector('.tab-btn[data-tab="execute"]').click()
  } finally {
    loadingOverlay.style.display = "none"
    allInputsAndButtons.forEach((el) => (el.disabled = false))
  }
}

function renderResults(data) {
  const resultsHeader = document.getElementById("resultsHeader")
  const statusIndicator = document.getElementById("statusIndicator")
  const resultsContent = document.getElementById("resultsContent")

  resultsHeader.style.display = "flex"
  resultsContent.innerHTML = ""

  statusIndicator.textContent = data.status
  statusIndicator.className = "status-indicator"
  if (data.status === "success") {
    statusIndicator.classList.add("status-success")
  } else if (data.status === "completed_with_errors" || data.status === "failed") {
    statusIndicator.classList.add("status-failed")
  } else {
    statusIndicator.classList.add("status-incomplete")
  }

  if (!data.process_step || data.process_step.length === 0) {
    resultsContent.innerHTML = `
            <div class="empty-state">
                <div class="empty-icon">📊</div>
                <h4>No steps were processed</h4>
                <p>The translation completed but no step results were returned</p>
            </div>
        `
    return
  }

  data.process_step.forEach((step, index) => {
    const stepEl = document.createElement("div")
    stepEl.className = "step-result"

    const stepStatusClass = `status-${step.status.toLowerCase().replace(/\s+/g, "-")}`

    stepEl.innerHTML = `
            <div class="step-header">
                <span class="step-name">${escapeHtml(step.step)}</span>
                <span class="status-indicator ${stepStatusClass}">${escapeHtml(step.status)}</span>
            </div>
            <div class="step-content">
                <div class="step-field primary-field">
                    <label>Input</label>
                    <div class="content">${step.input ? escapeHtml(step.input) : '<span class="no-data">No input provided</span>'}</div>
                </div>
                <div class="step-field primary-field">
                    <label>Output</label>
                    <div class="content">${step.output ? escapeHtml(step.output) : '<span class="no-data">No output generated</span>'}</div>
                </div>
                ${
                  step.prompt
                    ? `
                <div class="step-field prompt-field">
                    <button class="prompt-toggle" onclick="togglePrompt(${index})">
                        <span class="toggle-icon">▶</span>
                        View Prompt
                    </button>
                    <div class="prompt-content" id="prompt-${index}" style="display: none;">
                        <div class="content">${escapeHtml(step.prompt)}</div>
                    </div>
                </div>
                `
                    : ""
                }
            </div>
        `
    resultsContent.appendChild(stepEl)
  })
}

function togglePrompt(index) {
  const promptContent = document.getElementById(`prompt-${index}`)
  const toggleIcon = document.querySelector(`button[onclick="togglePrompt(${index})"] .toggle-icon`)

  if (promptContent.style.display === "none") {
    promptContent.style.display = "block"
    toggleIcon.textContent = "▼"
    toggleIcon.parentElement.innerHTML = '<span class="toggle-icon">▼</span>Hide Prompt'
  } else {
    promptContent.style.display = "none"
    toggleIcon.textContent = "▶"
    toggleIcon.parentElement.innerHTML = '<span class="toggle-icon">▶</span>View Prompt'
  }
}

function escapeHtml(unsafe) {
  if (typeof unsafe !== "string") {
    return String(unsafe)
  }
  return unsafe
    .replace(/&/g, "&amp;")
    .replace(/</g, "&lt;")
    .replace(/>/g, "&gt;")
    .replace(/"/g, "&quot;")
    .replace(/'/g, "&#039;")
}

function copyResults() {
  if (!lastTranslationResponse) {
    alert("No results to copy.")
    return
  }

  navigator.clipboard
    .writeText(JSON.stringify(lastTranslationResponse, null, 2))
    .then(() => {
      alert("Full response copied to clipboard!")
    })
    .catch((err) => {
      console.error("Failed to copy: ", err)
      alert("Could not copy results to clipboard.")
    })
}

function showWorkflowVisualization() {
  if (!lastTranslationResponse || !lastTranslationResponse.data) {
    alert("No workflow data available to visualize.")
    return
  }

  let workflowData = null

  if (lastTranslationResponse.data.process_step) {
    const workflowTranslationStep = lastTranslationResponse.data.process_step.find(
      (step) => step.step === "Workflow Translation",
    )

    if (workflowTranslationStep && workflowTranslationStep.output) {
      let workflowText = workflowTranslationStep.output

      workflowText = workflowText.replace(/```json\s*/g, "").replace(/```\s*/g, "")

      try {
        workflowData = JSON.parse(workflowText)
      } catch (e) {
        console.error("Failed to parse n8n workflow JSON:", e)
        alert("Unable to parse n8n workflow data for visualization.")
        return
      }
    }
  }

  if (!workflowData || !workflowData.nodes) {
    alert("No n8n workflow data available to visualize.")
    return
  }

  const modal = document.getElementById("workflowVisualizationModal")
  modal.style.display = "flex"

  createEnhancedN8nWorkflowVisualization(workflowData)
}

function hideWorkflowVisualizationModal() {
  document.getElementById("workflowVisualizationModal").style.display = "none"
}

function createEnhancedN8nWorkflowVisualization(workflowData) {
  const container = document.getElementById("workflowNetwork")

  // Add visualization controls if they don't exist
  if (!document.getElementById("visualizerControls")) {
    const controlsHtml = `
      <div id="visualizerControls" style="position: absolute; top: 10px; right: 10px; z-index: 1000; display: flex; gap: 8px; flex-wrap: wrap;">
        <button id="darkModeToggle" class="viz-control-btn" title="Toggle Dark Mode">🌙</button>
        <button id="fitViewBtn" class="viz-control-btn" title="Fit to View">📐</button>
        <button id="resetLayoutBtn" class="viz-control-btn" title="Reset Layout">🔄</button>
        <button id="togglePhysicsBtn" class="viz-control-btn" title="Toggle Physics">⚡</button>
        <button id="exportImageBtn" class="viz-control-btn" title="Export as Image">📷</button>
      </div>
      <div id="visualizerStats" style="position: absolute; bottom: 10px; left: 10px; z-index: 1000; background: rgba(0,0,0,0.7); color: white; padding: 8px 12px; border-radius: 4px; font-size: 12px;">
        <span id="nodeCount">Nodes: ${workflowData.nodes.length}</span> | 
        <span id="edgeCount">Connections: 0</span>
      </div>
    `
    container.insertAdjacentHTML("beforebegin", controlsHtml)
  }

  const nodes = []
  const edges = []
  const isDarkMode = localStorage.getItem("visualizerDarkMode") === "true"

  // Enhanced node creation with node_name field support
  workflowData.nodes.forEach((node, index) => {
    const displayName = node.node_name || extractSkillFromTask(node.name)
    const realName = node.name

    let x, y
    if (node.position && node.position.length >= 2) {
      x = node.position[0]
      y = node.position[1]
    } else {
      x = index * 280 + 150
      y = 200 + (index % 3) * 120
    }

    // Enhanced node styling with type-based colors
    const nodeColor = getNodeColor(node.type, isDarkMode)

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
      title: `<div style="max-width: 350px; white-space: normal; line-height: 1.5; font-family: Inter, sans-serif;">
        <strong style="color: ${nodeColor.background};">Real Name:</strong><br/>
        <span style="font-size: 12px;">${realName}</span><br/><br/>
        <strong>Display Name:</strong> ${displayName}<br/>
        <strong>Type:</strong> ${node.type}<br/>
        <strong>ID:</strong> ${node.id}
      </div>`,
      x: x,
      y: y,
      fixed: { x: false, y: false }, // Allow dragging
      borderWidth: 2,
      borderWidthSelected: 3,
      margin: 12,
    })
  })

  // Enhanced edge creation supporting multiple connections
  let edgeCount = 0
  if (workflowData.connections) {
    Object.keys(workflowData.connections).forEach((sourceNodeName) => {
      const connections = workflowData.connections[sourceNodeName]
      const sourceNode = workflowData.nodes.find((n) => n.name === sourceNodeName)

      if (!sourceNode) return

      // Handle main connections
      if (connections.main && Array.isArray(connections.main)) {
        connections.main.forEach((outputArray, outputIndex) => {
          if (Array.isArray(outputArray)) {
            outputArray.forEach((connection, connIndex) => {
              const targetNode = workflowData.nodes.find((n) => n.name === connection.node)
              if (targetNode) {
                edges.push({
                  id: `${sourceNode.id}-${targetNode.id}-${outputIndex}-${connIndex}`,
                  from: sourceNode.id,
                  to: targetNode.id,
                  arrows: { to: { enabled: true, scaleFactor: 1.2 } },
                  color: {
                    color: isDarkMode ? "#64b5f6" : "#1976d2",
                    highlight: isDarkMode ? "#90caf9" : "#1565c0",
                  },
                  width: 2,
                  smooth: { type: "cubicBezier", forceDirection: "horizontal", roundness: 0.4 },
                  label: outputIndex > 0 ? `out${outputIndex}` : "",
                  font: { size: 10, color: isDarkMode ? "#ffffff" : "#666666" },
                })
                edgeCount++
              }
            })
          }
        })
      }
    })
  }

  // Update edge count in stats
  const edgeCountElement = document.getElementById("edgeCount")
  if (edgeCountElement) {
    edgeCountElement.textContent = `Connections: ${edgeCount}`
  }

  const options = {
    layout: {
      hierarchical: {
        enabled: false,
      },
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
      shadow: isDarkMode
        ? {
            enabled: true,
            color: "rgba(0,0,0,0.5)",
            size: 10,
            x: 2,
            y: 2,
          }
        : {
            enabled: true,
            color: "rgba(0,0,0,0.2)",
            size: 8,
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
  }

  // Apply dark mode styling to container
  if (isDarkMode) {
    container.style.background = "linear-gradient(135deg, #1a1a1a 0%, #2d2d2d 100%)"
    container.style.border = "1px solid #404040"
  } else {
    container.style.background = "linear-gradient(135deg, #f8f9fa 0%, #e9ecef 100%)"
    container.style.border = "1px solid #dee2e6"
  }

  const networkData = {
    nodes: new vis.DataSet(nodes),
    edges: new vis.DataSet(edges),
  }

  const network = new vis.Network(container, networkData, options)

  // Enhanced event handlers
  setupVisualizerControls(network, networkData, isDarkMode)

  // Auto-fit on load
  setTimeout(() => {
    network.fit({ animation: { duration: 1000, easingFunction: "easeInOutQuad" } })
  }, 500)
}

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
  }

  return (
    colorMap[nodeType] ||
    (isDarkMode ? { background: "#607d8b", border: "#455a64" } : { background: "#78909c", border: "#607d8b" })
  )
}

function setupVisualizerControls(network, networkData, initialDarkMode) {
  let isDarkMode = initialDarkMode
  let physicsEnabled = true

  // Dark mode toggle
  const darkModeBtn = document.getElementById("darkModeToggle")
  if (darkModeBtn) {
    darkModeBtn.textContent = isDarkMode ? "☀️" : "🌙"
    darkModeBtn.onclick = () => {
      isDarkMode = !isDarkMode
      localStorage.setItem("visualizerDarkMode", isDarkMode.toString())
      darkModeBtn.textContent = isDarkMode ? "☀️" : "🌙"

      // Update container styling
      const container = document.getElementById("workflowNetwork")
      if (isDarkMode) {
        container.style.background = "linear-gradient(135deg, #1a1a1a 0%, #2d2d2d 100%)"
        container.style.border = "1px solid #404040"
      } else {
        container.style.background = "linear-gradient(135deg, #f8f9fa 0%, #e9ecef 100%)"
        container.style.border = "1px solid #dee2e6"
      }

      // Update node colors
      const nodes = networkData.nodes.get()
      nodes.forEach((node) => {
        const newColor = getNodeColor(node.type || "default", isDarkMode)
        networkData.nodes.update({
          id: node.id,
          color: newColor,
          font: { ...node.font, color: isDarkMode ? "#ffffff" : "#333333" },
        })
      })
    }
  }

  // Fit view
  const fitBtn = document.getElementById("fitViewBtn")
  if (fitBtn) {
    fitBtn.onclick = () => {
      network.fit({ animation: { duration: 800, easingFunction: "easeInOutQuad" } })
    }
  }

  // Reset layout
  const resetBtn = document.getElementById("resetLayoutBtn")
  if (resetBtn) {
    resetBtn.onclick = () => {
      network.setData(networkData)
      setTimeout(() => {
        network.fit({ animation: { duration: 1000, easingFunction: "easeInOutQuad" } })
      }, 100)
    }
  }

  // Toggle physics
  const physicsBtn = document.getElementById("togglePhysicsBtn")
  if (physicsBtn) {
    physicsBtn.onclick = () => {
      physicsEnabled = !physicsEnabled
      network.setOptions({ physics: { enabled: physicsEnabled } })
      physicsBtn.style.opacity = physicsEnabled ? "1" : "0.5"
      physicsBtn.title = physicsEnabled ? "Disable Physics" : "Enable Physics"
    }
  }

  // Export image
  const exportBtn = document.getElementById("exportImageBtn")
  if (exportBtn) {
    exportBtn.onclick = () => {
      const canvas = document.querySelector("#workflowNetwork canvas")
      if (canvas) {
        const link = document.createElement("a")
        link.download = "workflow-visualization.png"
        link.href = canvas.toDataURL()
        link.click()
      }
    }
  }
}

function downloadVisualization() {
  const resultsContent = document.getElementById("resultsContent")
  if (!resultsContent) {
    alert("No results available to visualize.")
    return
  }

  // Find n8n workflow data in results
  let workflowData = null
  const stepResults = resultsContent.querySelectorAll(".step-result")

  for (const stepEl of stepResults) {
    const outputContent = stepEl.querySelector(".step-content .content")
    if (outputContent) {
      const outputText = outputContent.textContent.trim()

      // Try to find JSON workflow data
      const jsonMatch = outputText.match(/\{[\s\S]*"nodes"[\s\S]*\}/)
      if (jsonMatch) {
        try {
          const workflowText = jsonMatch[0]
          workflowData = JSON.parse(workflowText)
          break
        } catch (e) {
          console.error("Failed to parse n8n workflow JSON:", e)
          continue
        }
      }
    }
  }

  if (!workflowData || !workflowData.nodes) {
    alert("No n8n workflow data available to download.")
    return
  }

  const htmlContent = createStandaloneVisualization(workflowData)

  const jsContent = `// Workflow Visualization - Standalone
// Generated on ${new Date().toISOString()}
// 
// To run this visualization:
// 1. Save this file with .js extension
// 2. Run: node thisfile.js
// 3. Or open the generated HTML file in a browser

const fs = require('fs');
const path = require('path');

const htmlContent = \`${htmlContent.replace(/`/g, "\\`").replace(/\$/g, "\\$")}\`;

// Write HTML file
const outputPath = path.join(__dirname, 'workflow-visualization.html');
fs.writeFileSync(outputPath, htmlContent, 'utf8');

console.log('Workflow visualization saved to:', outputPath);
console.log('Open the HTML file in your browser to view the visualization.');

// If running in browser environment, create blob and download
if (typeof window !== 'undefined') {
  const blob = new Blob([htmlContent], { type: 'text/html' });
  const url = URL.createObjectURL(blob);
  const a = document.createElement('a');
  a.href = url;
  a.download = 'workflow-visualization.html';
  document.body.appendChild(a);
  a.click();
  document.body.removeChild(a);
  URL.revokeObjectURL(url);
}
`

  // Download the JS file
  const blob = new Blob([jsContent], { type: "application/javascript" })
  const url = URL.createObjectURL(blob)
  const a = document.createElement("a")
  a.href = url
  a.download = "workflow-visualization-generator.js"
  document.body.appendChild(a)
  a.click()
  document.body.removeChild(a)
  URL.revokeObjectURL(url)

  // Also create direct HTML download as fallback
  const htmlBlob = new Blob([htmlContent], { type: "text/html" })
  const htmlUrl = URL.createObjectURL(htmlBlob)
  const htmlLink = document.createElement("a")
  htmlLink.href = htmlUrl
  htmlLink.download = "workflow-visualization.html"
  document.body.appendChild(htmlLink)
  htmlLink.click()
  document.body.removeChild(htmlLink)
  URL.revokeObjectURL(htmlUrl)

  alert("Visualization files downloaded! You now have both a JS generator and direct HTML file.")
}

function createStandaloneVisualization(workflowData) {
  return `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Workflow Visualizer</title>
    <script type="text/javascript" src="https://unpkg.com/vis-network/standalone/umd/vis-network.min.js"></script>
    <style>
        body {
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
            margin: 0;
            padding: 20px;
            background: #f5f5f5;
        }
        .container {
            max-width: 1200px;
            margin: 0 auto;
            background: white;
            border-radius: 8px;
            padding: 20px;
            box-shadow: 0 2px 10px rgba(0,0,0,0.1);
        }
        h1 {
            margin: 0 0 20px 0;
            color: #333;
        }
        #network {
            width: 100%;
            height: 600px;
            border: 1px solid #ddd;
            border-radius: 4px;
        }
        .info {
            margin-top: 15px;
            padding: 10px;
            background: #f8f9fa;
            border-radius: 4px;
            font-size: 14px;
            color: #666;
        }
    </style>
</head>
<body>
    <div class="container">
        <h1>Workflow Visualization</h1>
        <div id="network"></div>
        <div class="info">
            <strong>Instructions:</strong> Drag nodes to rearrange, zoom with mouse wheel, click and drag to pan. Hover over nodes to see details.
        </div>
    </div>

    <script>
        const workflowData = ${JSON.stringify(workflowData, null, 2)};
        
        function getStatusColor(status) {
            const statusLower = status.toLowerCase();
            
            if (statusLower.includes('success') || statusLower.includes('completed')) {
                return { bg: '#4CAF50', border: '#45a049', text: 'white' };
            } else if (statusLower.includes('error') || statusLower.includes('failed')) {
                return { bg: '#f44336', border: '#da190b', text: 'white' };
            } else if (statusLower.includes('warning')) {
                return { bg: '#ff9800', border: '#f57c00', text: 'white' };
            } else {
                return { bg: '#2196F3', border: '#1976D2', text: 'white' };
            }
        }
        
        function createVisualization() {
            const container = document.getElementById('network');
            
            const nodes = [];
            const edges = [];
            
            workflowData.nodes.forEach((node, index) => {
                let x, y
                if (node.position && node.position.length >= 2) {
                    x = node.position[0]
                    y = node.position[1]
                } else {
                    x = index * 280 + 150
                    y = 200 + (index % 3) * 120
                }
                
                nodes.push({
                    id: node.id,
                    label: node.name,
                    shape: 'box',
                    color: { 
                        background: '#2196f3', 
                        border: '#1976d2' 
                    },
                    font: { color: 'white' },
                    title: \`Type: \${node.type}\\nName: \${node.name}\`,
                    x: x,
                    y: y,
                    fixed: { x: false, y: false },
                    borderWidth: 2,
                    borderWidthSelected: 3,
                    margin: 12
                });
            });
            
            if (workflowData.connections) {
                Object.keys(workflowData.connections).forEach(sourceNodeName => {
                    const connections = workflowData.connections[sourceNodeName];
                    
                    if (connections.main && Array.isArray(connections.main)) {
                        connections.main.forEach((outputArray, outputIndex) => {
                            if (Array.isArray(outputArray)) {
                                outputArray.forEach((connection, connIndex) => {
                                    const targetNode = workflowData.nodes.find(n => n.name === connection.node)
                                    if (targetNode) {
                                        edges.push({
                                            id: \`\${sourceNode.id}-\${targetNode.id}-\${outputIndex}-\${connIndex}\`,
                                            from: sourceNode.id,
                                            to: targetNode.id,
                                            arrows: { to: { enabled: true, scaleFactor: 1.2 } },
                                            color: { 
                                                color: '#64b5f6',
                                                highlight: '#90caf9'
                                            },
                                            width: 2,
                                            smooth: { type: 'cubicBezier', forceDirection: 'horizontal', roundness: 0.4 },
                                            label: outputIndex > 0 ? \`out\${outputIndex}\` : "",
                                            font: { size: 10, color: '#666666' }
                                        })
                                    }
                                })
                            }
                        })
                    }
                });
            }
            
            const data = { nodes: new vis.DataSet(nodes), edges: new vis.DataSet(edges) };
            const options = {
                layout: {
                    hierarchical: {
                        enabled: false
                    }
                },
                physics: {
                    enabled: true,
                    stabilization: { iterations: 100 },
                    barnesHut: {
                        gravitationalConstant: -8000,
                        centralGravity: 0.3,
                        springLength: 200,
                        springConstant: 0.04,
                        damping: 0.09
                    }
                },
                interaction: {
                    dragNodes: true,
                    dragView: true,
                    zoomView: true,
                    hover: true,
                    selectConnectedEdges: false,
                    tooltipDelay: 200
                },
                nodes: {
                    margin: 15,
                    font: { size: 13 },
                    widthConstraint: { minimum: 140, maximum: 220 },
                    heightConstraint: { minimum: 40 },
                    shadow: {
                        enabled: true,
                        color: 'rgba(0,0,0,0.2)',
                        size: 8,
                        x: 2,
                        y: 2
                    }
                },
                edges: {
                    smooth: { 
                        type: "cubicBezier", 
                        forceDirection: "horizontal",
                        roundness: 0.4
                    },
                    width: 2,
                    shadow: {
                        enabled: true,
                        color: 'rgba(0,0,0,0.2)',
                        size: 5,
                        x: 1,
                        y: 1
                    }
                }
            };
            
            const network = new vis.Network(container, data, options)
            
            // Auto-fit on load
            setTimeout(() => {
                network.fit({ animation: { duration: 1000, easingFunction: "easeInOutQuad" } })
            }, 500)
        }
        
        document.addEventListener('DOMContentLoaded', createVisualization);
    </script>
</body>
</html>`
}

document.addEventListener("keypress", (e) => {
  if (e.key === "Enter") {
    if (e.target.id === "taskMarkdown") {
      return
    } else if (e.target.id === "agentSkillsBulk" || e.target.id === "globalSkillsBulk") {
      return
    }
  }
})

async function showPromptVersions(promptName) {
  const modal = document.getElementById("promptVersionsModal")
  const title = document.getElementById("versionsModalTitle")
  const versionsList = document.getElementById("versionsList")
  const templateContent = document.getElementById("templateContent")
  const selectBtn = document.getElementById("selectVersionBtn")

  title.textContent = `${promptName} - Versions`
  versionsList.innerHTML = '<div class="mini-spinner"></div><span>Loading versions...</span>'
  templateContent.textContent = "Select a version to view its template"
  templateContent.className = "template-content"
  selectBtn.disabled = true
  selectedVersionId = null

  modal.style.display = "flex"

  try {
    const response = await fetch(`${BASE_URL}/prompts/${encodeURIComponent(promptName)}`)
    const data = await response.json()

    console.log("[v0] API Response:", data)

    if (response.ok && data.data.prompt_name && data.data.versions) {
      currentPromptVersions = data.data
      renderVersionsList(data.data.versions)

      if (data.data.versions.length > 0) {
        selectVersion(data.data.versions[0])
      }
    } else {
      console.log("[v0] Response not in expected format:", data)
      versionsList.innerHTML = `
        <div style="padding: 20px; text-align: center; color: var(--accent-danger);">
          Failed to load versions: ${data.message || "Unexpected response format"}
        </div>
      `
    }
  } catch (error) {
    console.error("Error loading prompt versions:", error)
    versionsList.innerHTML = `
      <div style="padding: 20px; text-align: center; color: var(--accent-danger);">
        Error loading versions. Please try again.
      </div>
    `
  }
}

function renderVersionsList(versions) {
  const versionsList = document.getElementById("versionsList")
  versionsList.innerHTML = ""

  versions.forEach((version, index) => {
    const versionElement = document.createElement("div")
    versionElement.className = "version-item"
    versionElement.onclick = (event) => selectVersion(version, event.currentTarget)

    const isLatest = index === 0
    const createdDate = new Date(version.created_at).toLocaleDateString()

    versionElement.innerHTML = `
      <div class="version-info">
        <div class="version-number">Version ${version.version}</div>
        <div class="version-date">${createdDate}</div>
      </div>
      ${isLatest ? '<span class="version-latest">Latest</span>' : ""}
    `

    versionsList.appendChild(versionElement)
  })
}

function selectVersion(version, eventTarget = null) {
  document.querySelectorAll(".version-item").forEach((item) => {
    item.classList.remove("selected")
  })

  if (eventTarget) {
    eventTarget.classList.add("selected")
  } else {
    const versionElements = document.querySelectorAll(".version-item")
    versionElements.forEach((element, index) => {
      if (currentPromptVersions && currentPromptVersions.versions[index].id === version.id) {
        element.classList.add("selected")
      }
    })
  }

  const templateContent = document.getElementById("templateContent")
  templateContent.textContent = version.template
  templateContent.className = "template-content has-content"

  selectedVersionId = version.id

  document.getElementById("selectVersionBtn").disabled = false
}

function selectPromptVersion() {
  if (!currentPromptVersions || !selectedVersionId) return

  const selectedVersion = currentPromptVersions.versions.find((v) => v.id === selectedVersionId)
  if (!selectedVersion) return

  const stepData = {
    name: currentPromptVersions.prompt_name,
    version_id: selectedVersionId,
    version: selectedVersion.version,
  }

  const existingIndex = selectedSteps.findIndex((step) => step.name === stepData.name)
  if (existingIndex !== -1) {
    selectedSteps[existingIndex] = stepData
  } else {
    selectedSteps.push(stepData)
  }

  updatePipelineStepsDisplay()
  hidePromptVersionsModal()
}

function hidePromptVersionsModal() {
  document.getElementById("promptVersionsModal").style.display = "none"
  currentPromptVersions = null
  selectedVersionId = null
}

function switchTab(tabName) {
  const tabBtns = document.querySelectorAll(".tab-btn")
  const tabContents = document.querySelectorAll(".tab-content")

  tabBtns.forEach((btn) => {
    btn.classList.remove("active")
  })

  tabContents.forEach((content) => {
    content.classList.remove("active")
  })

  document.querySelector(`.tab-btn[data-tab="${tabName}"]`).classList.add("active")
  document.getElementById(`${tabName}-tab`).classList.add("active")
}

function extractSkillFromTask(taskName) {
  const skillPatterns = [
    { pattern: /fetch.*google sheets/i, skill: "Google Sheets" },
    { pattern: /summarize.*metrics/i, skill: "Data Analysis" },
    { pattern: /generate.*email/i, skill: "Email Writing" },
    { pattern: /send.*gmail/i, skill: "Gmail" },
    { pattern: /transcribe.*video/i, skill: "Video Transcription" },
    { pattern: /write.*copy/i, skill: "Copywriting" },
    { pattern: /analyze.*data/i, skill: "Data Analysis" },
    { pattern: /create.*report/i, skill: "Report Generation" },
    { pattern: /process.*image/i, skill: "Image Processing" },
    { pattern: /translate.*text/i, skill: "Translation" },
    { pattern: /schedule.*meeting/i, skill: "Scheduling" },
    { pattern: /upload.*file/i, skill: "File Upload" },
    { pattern: /download.*data/i, skill: "Data Download" },
    { pattern: /convert.*format/i, skill: "Format Conversion" },
    { pattern: /validate.*input/i, skill: "Data Validation" },
  ]

  for (const { pattern, skill } of skillPatterns) {
    if (pattern.test(taskName)) {
      return skill
    }
  }

  const words = taskName.split(" ").slice(0, 2)
  return words.map((word) => word.charAt(0).toUpperCase() + word.slice(1).toLowerCase()).join(" ")
}
