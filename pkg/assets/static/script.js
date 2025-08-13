// Global variables
let tasks = [];
let agentSkills = [];
let globalSkills = [];
let availableSteps = [];
let selectedSteps = [];
let lastTranslationResponse = null;

// Base URL - modify this to match your API
const BASE_URL = 'http://localhost:8019/api/v1'; // Change this to your API URL

// Tab functionality
document.addEventListener('DOMContentLoaded', function() {
    initializeTabs();
    loadPipelineSteps();
    addTask(); // Add initial empty task
});

function initializeTabs() {
    const tabBtns = document.querySelectorAll('.tab-btn');
    const tabContents = document.querySelectorAll('.tab-content');

    tabBtns.forEach(btn => {
        btn.addEventListener('click', () => {
            const targetTab = btn.getAttribute('data-tab');
            
            // Remove active class from all tabs and contents
            tabBtns.forEach(b => b.classList.remove('active'));
            tabContents.forEach(c => c.classList.remove('active'));
            
            // Add active class to clicked tab and corresponding content
            btn.classList.add('active');
            document.getElementById(`${targetTab}-tab`).classList.add('active');
            
            // Update summaries when switching to execute tab
            if (targetTab === 'execute') {
                updateExecutionSummary();
            }
        });
    });
}

// Task management functions
function addTask() {
    const taskList = document.getElementById('taskList');
    const taskIndex = tasks.length;
    
    const taskItem = document.createElement('div');
    taskItem.className = 'task-item';
    taskItem.innerHTML = `
        <input type="text" placeholder="Enter task description..." 
                onchange="updateTask(${taskIndex}, this.value)" 
                onkeyup="updateTask(${taskIndex}, this.value)">
        <button class="remove-task-btn" onclick="removeTask(${taskIndex})">Remove</button>
    `;
    
    taskList.appendChild(taskItem);
    tasks.push('');
}

function removeTask(index) {
    tasks.splice(index, 1);
    renderTasks();
}

function updateTask(index, value) {
    if (index < tasks.length) {
        tasks[index] = value;
    }
}

function renderTasks() {
    const taskList = document.getElementById('taskList');
    taskList.innerHTML = '';
    
    tasks.forEach((task, index) => {
        const taskItem = document.createElement('div');
        taskItem.className = 'task-item';
        taskItem.innerHTML = `
            <input type="text" value="${task}" placeholder="Enter task description..." 
                    onchange="updateTask(${index}, this.value)" 
                    onkeyup="updateTask(${index}, this.value)">
            <button class="remove-task-btn" onclick="removeTask(${index})">Remove</button>
        `;
        taskList.appendChild(taskItem);
    });
}

// Skills management functions
function addSkill(type) {
    const input = document.getElementById(`${type}SkillInput`);
    const skill = input.value.trim();
    
    if (skill) {
        if (type === 'agent') {
            agentSkills.push(skill);
        } else {
            globalSkills.push(skill);
        }
         
        input.value = '';
        renderSkills(type);
    }
}

function addSkillsBulk(type) {
    const textarea = document.getElementById(`${type}SkillsBulk`);
    const skills = textarea.value.split('\n').map(s => s.trim()).filter(s => s);
    
    if (skills.length > 0) {
        if (type === 'agent') {
            agentSkills.push(...skills);
        } else {
            globalSkills.push(...skills);
        }
        
        textarea.value = '';
        renderSkills(type);
    }
}

function removeSkill(type, index) {
    if (type === 'agent') {
        agentSkills.splice(index, 1);
    } else {
        globalSkills.splice(index, 1);
    }
    renderSkills(type);
}

function renderSkills(type) {
    const container = document.getElementById(`${type}Skills`);
    const skillsArray = type === 'agent' ? agentSkills : globalSkills;
    
    container.innerHTML = '';
    
    skillsArray.forEach((skill, index) => {
        const skillTag = document.createElement('div');
        skillTag.className = 'skill-tag';
        skillTag.innerHTML = `
            ${skill}
            <span class="remove-skill" onclick="removeSkill('${type}', ${index})">×</span>
        `;
        container.appendChild(skillTag);
    });
}

// Pipeline steps functions
async function loadPipelineSteps() {
    const loadingIndicator = document.getElementById('pipelineLoading');
    loadingIndicator.style.display = 'flex';
    
    try {
        const response = await fetch(`${BASE_URL}/prompts/steps`);
        const data = await response.json();
        
        if (data.status === 'success') {
            availableSteps = data.data;
            renderPipelineSteps();
        } else {
            console.error('Failed to load pipeline steps:', data.message);
            // Show error message to user
            const container = document.getElementById('pipelineSteps');
            container.innerHTML = `<div style="padding: 20px; text-align: center; color: #dc3545;">Failed to load steps: ${data.message}</div>`;
        }
    } catch (error) {
        console.error('Error loading pipeline steps:', error);
        // Show error message to user
        const container = document.getElementById('pipelineSteps');
        container.innerHTML = `<div style="padding: 20px; text-align: center; color: #dc3545;">Error loading steps. Please check your connection and try again.</div>`;
    } finally {
        // Hide loading indicator
        loadingIndicator.style.display = 'none';
    }
}

// Function to refresh pipeline steps
function refreshPipelineSteps() {
    // Clear current selections when refreshing
    selectedSteps = [];
    loadPipelineSteps();
    renderPipelineOrder();
}

// Update the renderPipelineSteps function in script.js
function renderPipelineSteps() {
    const container = document.getElementById('pipelineSteps');
    
    // Create steps container without the old refresh button
    container.innerHTML = `<div id="stepsGrid"></div>`;
    
    const stepsGrid = document.getElementById('stepsGrid');
    stepsGrid.style.display = 'grid';
    stepsGrid.style.gridTemplateColumns = 'repeat(auto-fit, minmax(250px, 1fr))';
    stepsGrid.style.gap = '15px';
    
    availableSteps.forEach(step => {
        const stepCard = document.createElement('div');
        stepCard.className = 'step-card';
        stepCard.onclick = () => toggleStep(step);
        
        if (selectedSteps.some(s => s.id === step.id)) {
            stepCard.classList.add('selected');
        }
        
        stepCard.innerHTML = `
            <h4>${step.name}</h4>
            <p>Version ${step.version}</p>
            <p>Created: ${new Date(step.created_at).toLocaleDateString()}</p>
        `;
        
        stepsGrid.appendChild(stepCard);
    });
}

// Update the refreshPipelineSteps function to show loading state on button
async function refreshPipelineSteps() {
    const refreshBtn = document.querySelector('.refresh-steps-btn');
    const refreshIcon = document.querySelector('.refresh-icon');
    const loadingIndicator = document.getElementById('pipelineLoading');
    
    // Show loading state
    refreshBtn.disabled = true;
    refreshIcon.style.animation = 'spin 1s linear infinite';
    loadingIndicator.style.display = 'flex';
    
    // Clear current selections when refreshing
    selectedSteps = [];
    
    try {
        const response = await fetch(`${BASE_URL}/prompts/steps`);
        const data = await response.json();
        console.log(data)
        
        if (data.status === 'success') {
            availableSteps = data.data;
            renderPipelineSteps();
        } else {
            console.error('Failed to load pipeline steps:', data.message);
            // Show error message to user
            const container = document.getElementById('pipelineSteps');
            container.innerHTML = `<div style="padding: 20px; text-align: center; color: #dc3545;">Failed to refresh steps: ${data.message}</div>`;
        }
    } catch (error) {
        // console.error('Error refreshing pipeline steps:', error);
        // // Show error message to user
        // const container = document.getElementById('pipelineSteps');
        // container.innerHTML = `<div style="padding: 20px; text-align: center; color: #dc3545;">Error refreshing steps. Please check your connection and try again.</div>`;
    } finally {
        // Restore button state and hide loading
        refreshBtn.disabled = false;
        refreshIcon.style.animation = 'none';
        loadingIndicator.style.display = 'none';
    }
    
    renderPipelineOrder();
}

function toggleStep(step) {
    const index = selectedSteps.findIndex(s => s.id === step.id);
    
    if (index === -1) {
        selectedSteps.push(step);
    } else {
        selectedSteps.splice(index, 1);
    }
    
    renderPipelineSteps();
    renderPipelineOrder();
}

function renderPipelineOrder() {
    const container = document.getElementById('orderList');
    
    if (selectedSteps.length === 0) {
        container.innerHTML = '<p class="empty-order">No steps selected</p>';
        return;
    }
    
    container.innerHTML = '';
    
    selectedSteps.forEach((step, index) => {
        const orderItem = document.createElement('div');
        orderItem.className = 'order-item';
        orderItem.innerHTML = `
            <div style="display: flex; align-items: center; gap: 10px;">
                <div class="order-number">${index + 1}</div>
                <span>${step.name}</span>
            </div>
            <button onclick="removeFromOrder(${index})" style="background: #dc3545; color: white; border: none; padding: 4px 8px; border-radius: 4px; cursor: pointer;">Remove</button>
        `;
        container.appendChild(orderItem);
    });
}

function removeFromOrder(index) {
    selectedSteps.splice(index, 1);
    renderPipelineSteps();
    renderPipelineOrder();
}

// Execution functions
function updateExecutionSummary() {
    document.getElementById('taskSummary').textContent = `${tasks.filter(t => t.trim()).length} tasks`;
    document.getElementById('agentSkillsSummary').textContent = `${agentSkills.length} skills`;
    document.getElementById('globalSkillsSummary').textContent = `${globalSkills.length} skills`;
    document.getElementById('stepsSummary').textContent = `${selectedSteps.length} steps`;
}

function validateTranslationData() {
    const errors = [];
    if (tasks.filter(t => t.trim()).length === 0) {
        errors.push('Please add at least one task.');
    }
    if (selectedSteps.length === 0) {
        errors.push('Please select at least one pipeline step.');
    }
    return errors;
}

function showValidationErrors(errors) {
    const errorContainer = document.getElementById('validationErrors');
    errorContainer.innerHTML = '<ul>' + errors.map(e => `<li>${e}</li>`).join('') + '</ul>';
    errorContainer.style.display = 'block';
}

function hideValidationErrors() {
    const errorContainer = document.getElementById('validationErrors');
    errorContainer.style.display = 'none';
}

async function performTranslation() {
    hideValidationErrors();
    const validationErrors = validateTranslationData();
    if (validationErrors.length > 0) {
        showValidationErrors(validationErrors);
        return;
    }

    const loadingOverlay = document.getElementById('loadingOverlay');
    const allInputsAndButtons = document.querySelectorAll('input, button, textarea, .step-card');

    loadingOverlay.style.display = 'flex';
    allInputsAndButtons.forEach(el => el.disabled = true);

    try {
        // Prepare request body with correct structure
        const requestBody = {
            task_list: tasks.filter(t => t.trim()).join('\n'), // Filter out empty tasks
            agent_skills: agentSkills,
            global_skills: globalSkills,
            steps: selectedSteps.map(s => s.name) // Extract step IDs
        };

        console.log('Sending translation request:', requestBody);

        const response = await fetch(`${BASE_URL}/translator`, {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json'
            },
            body: JSON.stringify(requestBody)
        });

        const result = await response.json();
        lastTranslationResponse = result;

        console.log('Translation response:', result);

        if (response.ok && result.status === 'success') {
            renderResults(result.data);
            // Automatically switch to results tab
            document.querySelector('.tab-btn[data-tab="results"]').click();
        } else {
            const error = result.message || 'An unknown error occurred.';
            showValidationErrors(['API Error: ' + error]);
            document.querySelector('.tab-btn[data-tab="execute"]').click();
        }

    } catch (error) {
        console.error('Translation Error:', error);
        showValidationErrors(['A network or system error occurred. Please check the console for more details.']);
        document.querySelector('.tab-btn[data-tab="execute"]').click();
    } finally {
        loadingOverlay.style.display = 'none';
        allInputsAndButtons.forEach(el => el.disabled = false);
    }
}

function renderResults(data) {
    const resultsHeader = document.getElementById('resultsHeader');
    const statusIndicator = document.getElementById('statusIndicator');
    const processStepsContainer = document.getElementById('processSteps');

    // Show header and clear previous results
    resultsHeader.style.display = 'flex';
    processStepsContainer.innerHTML = '';

    // Update overall status
    statusIndicator.textContent = data.status;
    statusIndicator.className = 'status-indicator'; // reset classes
    if (data.status === 'success') {
        statusIndicator.classList.add('status-success');
    } else if (data.status === 'completed_with_errors' || data.status === 'failed') {
        statusIndicator.classList.add('status-failed');
    } else {
        statusIndicator.classList.add('status-incomplete');
    }
    
    if (!data.process_step || data.process_step.length === 0) {
        processStepsContainer.innerHTML = '<div class="no-results"><p>No steps were processed.</p></div>';
        return;
    }

    // Render each step
    data.process_step.forEach((step, index) => {
        const stepEl = document.createElement('div');
        stepEl.className = 'step-result';

        const stepStatusClass = `status-${step.status.toLowerCase().replace(/\s+/g, '-')}`;

        stepEl.innerHTML = `
            <div class="step-header">
                <span>${escapeHtml(step.step)}</span>
                <span class="status-indicator ${stepStatusClass}">${escapeHtml(step.status)}</span>
            </div>
            <div class="step-content">
                <div class="step-field">
                    <label>Input</label>
                    <div class="content">${step.input ? `<pre>${escapeHtml(step.input)}</pre>` : '<span style="color: #6c757d; font-style: italic;">No input provided</span>'}</div>
                </div>
                <div class="step-field">
                    <label>Output</label>
                    <div class="content">${step.output ? `<pre>${escapeHtml(step.output)}</pre>` : '<span style="color: #6c757d; font-style: italic;">No output generated</span>'}</div>
                </div>
                ${step.prompt ? `
                <div class="step-field">
                    <label>Prompt</label>
                    <div class="content"><pre>${escapeHtml(step.prompt)}</pre></div>
                </div>
                ` : ''}
            </div>
        `;
        processStepsContainer.appendChild(stepEl);
    });
}

function escapeHtml(unsafe) {
    if (typeof unsafe !== 'string') {
        // Convert to string if it's not already
        return String(unsafe);
    }
    return unsafe
        .replace(/&/g, "&amp;")
        .replace(/</g, "&lt;")
        .replace(/>/g, "&gt;")
        .replace(/"/g, "&quot;")
        .replace(/'/g, "&#039;");
}

function copyResults() {
    if (!lastTranslationResponse) {
        alert('No results to copy.');
        return;
    }
    
    navigator.clipboard.writeText(JSON.stringify(lastTranslationResponse, null, 2))
        .then(() => {
            alert('Full response copied to clipboard!');
        })
        .catch(err => {
            console.error('Failed to copy: ', err);
            alert('Could not copy results to clipboard.');
        });
}