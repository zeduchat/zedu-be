package seed

import (
	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/utility"
	"gorm.io/gorm"
)

func SeedTranslatorPrompt(logger *utility.Logger, db *gorm.DB) {
	var count int64

	if err := db.Model(&models.Prompts{}).
		Where("name IN ?", []string{"Task Cleanup", "Skill Matching", "Workflow Translation"}).
		Count(&count).Error; err != nil {
		logger.Error("translator prompt seeding: " + err.Error())
		return
	}

	if count > 0 {
		logger.Error("Translator Prompt already exist, skipping seeding...")
		return
	}

	translatorPrompts := []models.Prompts{
		{
			ID:   utility.GenerateUUID(),
			Name: "Task Cleanup",
			Template: `You are a Task Cleaner. Your job is to take free-form Markdown task lists and convert them to clean, structured tasks.

Output: A JSON array of clean, actionable task descriptions. Each task should be one clear intent.

Rules:
- Remove markdown formatting (bullets, numbers)
- Make each task specific and actionable
- Combine related sub-tasks if needed
- Remove duplicates

Example:
Input: "- Pull leads from hubspot\n- send emails to marketing team\n- export the data"
Output: ["Pull new leads from HubSpot CRM", "Send email report to marketing team", "Export lead data to CSV format"]

Return only the raw JSON array exactly as specified. 
Do not include any explanations, preamble, notes, or extra text. 
Do not say anything before or after the JSON. 
Output must be a single valid JSON array.`,
			Version: 1,
		},
		{
			ID:   utility.GenerateUUID(),
			Name: "Skill Matching",
			Template: `You are a Skill Matcher.

Input:
- Agent skills: {{agent_skills}}
- Global skills (fallback): {{global_skills}}
- Cleaned tasks (user input).

Output:
- If all tasks have matching skills, return JSON array:
  [
    {"task": "description", "candidates": ["skill1", "skill2"]}
  ]

Rules:
- Match tasks to best skills.
- Prefer agent over global skills.
- Extract parameters when possible.
- Do NOT invent or force-fit agent skills.
- If a task has no suitable candidate, return instead:

  {
    "missing_skills": [list from global skills],
    "suggestion": "If none fit, consider integrating external service like [example]."
  }

- Return only valid JSON (either mapping array or missing_skills object).
- No explanations or extra text.
- Output must be a single JSON entity.

Be exact and strict.`,
			Version: 1,
		},
		{
			ID:   utility.GenerateUUID(),
			Name: "Workflow Translation",
			Template: `YOU ARE AN EXPERT N8N WORKFLOW COMPILER. CONVERT the given array of task objects INTO a VALID, SEQUENTIALLY CONNECTED N8N WORKFLOW JSON. OUTPUT ONLY the JSON.

### INPUT FORMAT ###
[
    {"task": "...", "candidates": ["skill_name1", "skill_name2"]},
    {"task": "...", "candidates": ["skill_name"]}
]

### RULES ###
1. Each task → one node (id, name, type, position, parameters).
2. Map candidates skills to correct n8n node type (skill_video_transcription → n8n-nodes-base.speechToText, skill_copywriting → n8n-nodes-base.aiWriter, etc.).
3. Connect sequentially: each node to the next; last has no outgoing.
4. Keep positions readable (x+200 each step).
5. Use task as node name; preserve all params exactly.
6. IDs as node-1, node-2, etc.

### CHAIN OF THOUGHT ###
1. UNDERSTAND tasks and skills.
2. MAP skills → node types.
3. BUILD nodes.
4. LINK sequentially.
5. VERIFY JSON validity.

### WHAT NOT TO DO ###
- NEVER output non-JSON.
- DO NOT change task wording or parameters.
- NO missing or extra nodes.
- NO malformed JSON.

### OUTPUT EXAMPLE ###
{
  "name": "Sample Workflow",
  "nodes": [
    {
      "id": "node-1",
      "name": "Trigger",
      "type": "n8n-nodes-base.manualTrigger",
      "position": [100, 200],
      "parameters": {}
    },
    {
      "id": "node-2",
      "name": "Processor",
      "type": "n8n-nodes-base.httpRequest",
      "position": [300, 200],
      "parameters": {}
    }
  ],
  "connections": {
    "Trigger": {
      "main": [
        [
          {
            "node": "Processor",
            "type": "main",
            "index": 0
          }
        ]
      ]
    }
  }
}`,
			Version: 1,
		},
	}

	db = db.Debug()
	for _, tp := range translatorPrompts {
		if err := db.Create(&tp).Error; err != nil {
			logger.Error("failed to seed translator_prompts: " + err.Error())
		}
	}
}
