package seed

import (
	"gorm.io/gorm"

	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/utility"
)

func SeedZeduResourcePages(logger *utility.Logger, db *gorm.DB) {
	pages := []models.ZeduResourcePage{
		{
			ID:       utility.GenerateUUID(),
			Title:    "How Bootcamps Centralize Learning and Communication in One Workspace",
			Overview: "For bootcamps, speed is everything. Learners move fast, instructors juggle multiple cohorts, and communication happens across countless touchpoints. But when tools are fragmented, learning slows down. One growing bootcamp reimagined its entire learning experience by consolidating everything into a single platform: Zedu.",
			Content: `The Challenge: Learning Was Scattered Across Too Many Tools

As the bootcamp scaled, its operations became increasingly complex:
- Conversations happened in chat apps
- Course content lived in separate platforms
- Assignments were tracked manually
- Mentor support was inconsistent

This fragmentation created friction for both learners and instructors. "Students didn't know where to ask questions, and mentors didn't have visibility into what was happening across the cohort."

The Solution: A Unified Learning Workspace with Zedu

Zedu replaced multiple disconnected tools with a single, structured workspace built around how bootcamps actually operate. The bootcamp leveraged:
- Channels to organize cohorts, modules, and discussions
- Group chats for team collaboration and peer learning
- 1:1 chats for mentor-student support
- File management for centralized learning resources
- Calls for live sessions, reviews, and mentoring
- Collaboration tools to keep everyone aligned in real time

Everything from communication to content now lived in one place.

The Impact: Faster Learning, Better Outcomes

With Zedu:
- Learners always knew where to go and what to do
- Mentors had full visibility into student progress and blockers
- Collaboration became natural and continuous

Results:
- Significant improvement in learner engagement
- Faster issue resolution during assignments
- Higher cohort completion rates

Why It Matters

Bootcamps thrive on momentum. By removing friction and centralizing workflows, Zedu helps learners stay focused and instructors stay effective.`,
		},
		{
			ID:       utility.GenerateUUID(),
			Title:    "How Universities Create a Connected Digital Campus Experience",
			Overview: "Universities are complex ecosystems. Students, faculty, and administrators interact daily but often through disconnected systems that slow everything down. Zedu enables institutions to bring everything together into a single, intelligent communication layer.",
			Content: `The Challenge: Disconnected Communication Across Campus

A university faced ongoing issues with:
- Missed announcements and scattered updates
- Repetitive student inquiries across departments
- Limited collaboration between faculty and students
- No centralized access to academic resources

"Information existed but accessing it was the real problem."

The Solution: Structured Communication Through Channels and Groups

With Zedu, the university created a digitally connected campus:
- Channels for departments, courses, and announcements
- Groups for project teams and study circles
- Chats for direct student-faculty communication
- File management for easy access to academic materials
- Calls for virtual lectures, office hours, and discussions
- Collaboration tools to streamline teamwork

This structure mirrored how the university operates — just more efficiently.

The Impact: A More Responsive and Engaged Campus

After implementation:
- Students accessed information faster and more easily
- Faculty reduced time spent on repetitive communication
- Collaboration across departments improved

Results:
- Increased student engagement
- Faster response times across campus
- More organized academic workflows

Why It Matters

A connected campus isn't just about communication — it's about creating an environment where students and educators can thrive. Zedu makes that possible.`,
		},
		{
			ID:       utility.GenerateUUID(),
			Title:    "How Schools Simplify Classroom Management and Student Collaboration",
			Overview: "In a school environment, simplicity and structure are critical. Teachers need tools that are easy to manage, while students need clear, focused spaces to learn and collaborate. Zedu brings both together.",
			Content: `The Challenge: Managing Classrooms in a Digital-First World

Teachers were dealing with:
- Multiple tools for communication and assignments
- Constant interruptions from repetitive questions
- Difficulty organizing class discussions and materials
- Limited visibility into student participation

"We had tools but no system."

The Solution: A Structured, Teacher-Controlled Classroom

Zedu enabled schools to build organized digital classrooms using:
- Channels for each class and subject
- Group spaces for student collaboration
- Chats for quick teacher-student interactions
- File management for lesson materials and assignments
- Calls for virtual teaching and check-ins
- Collaboration features to support interactive learning

Teachers could now manage everything from a single platform.

The Impact: More Focus, Less Chaos

With Zedu:
- Classrooms became more organized and predictable
- Students engaged more actively in discussions
- Teachers spent less time managing tools

Results:
- Improved assignment completion rates
- Better classroom participation
- Reduced administrative workload

Why It Matters

Technology should simplify teaching, not complicate it. Zedu gives teachers control, clarity, and time back.`,
		},
		{
			ID:       utility.GenerateUUID(),
			Title:    "How Institutions Enable Seamless Collaboration Across Learning Environments",
			Overview: "Whether it's group projects, faculty coordination, or cross-department initiatives, collaboration is at the heart of education. But without the right tools, it becomes inefficient and fragmented. Zedu redefines collaboration by making it seamless and structured.",
			Content: `The Challenge: Collaboration Without Structure

Across institutions, collaboration often looked like this:
- Files shared across multiple platforms
- Conversations lost in long message threads
- No clear ownership or visibility
- Meetings disconnected from ongoing work

"We were collaborating but not effectively."

The Solution: Integrated Communication and Collaboration

Zedu brought collaboration into a single ecosystem:
- Groups for project-based teamwork
- Channels for structured discussions
- File management for centralized document access
- Chats for quick decision-making
- Calls for real-time collaboration
- Built-in workflows to keep everything aligned

This ensured that communication and execution happened in the same place.

The Impact: Smarter, Faster Teamwork

With Zedu:
- Teams collaborated in real time with full context
- Files and discussions stayed organized and accessible
- Projects moved forward without unnecessary delays

Results:
- Faster project completion cycles
- Improved team alignment
- Reduced miscommunication

Why It Matters

Great collaboration isn't just about communication — it's about structure, visibility, and flow. Zedu brings all three together.`,
		},
	}

	for _, page := range pages {
		var existing models.ZeduResourcePage
		if err := db.Unscoped().Where("title = ?", page.Title).First(&existing).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				if err := db.Create(&page).Error; err != nil {
					logger.Error("failed to create zedu resource page: " + err.Error())
				}
			} else {
				logger.Error("failed to query zedu resource page: " + err.Error())
			}
		} else {
			page.ID = existing.ID
			page.DeletedAt = gorm.DeletedAt{Valid: false}
			if err := db.Unscoped().Model(&existing).Updates(page).Error; err != nil {
				logger.Error("failed to update zedu resource page: " + err.Error())
			}
		}
		logger.Info("Zedu resource pages seeded successfully>>>>>>>>>>>>>>")
	}
}

func SeedZeduGuides(logger *utility.Logger, db *gorm.DB) {
	guides := []models.ZeduGuide{
		{
			ID:    utility.GenerateUUID(),
			Title: "Organizing Courses with Channels",
			Content: `Managing multiple courses, topics, and discussions in a digital learning environment like Zedu can quickly become overwhelming, especially in large classrooms or fast-paced bootcamps. Without a clear structure, important conversations get buried, students struggle to follow discussions, and instructors find it difficult to manage engagement effectively.

This is where channels play a critical role.

On Zedu, channels are not just a feature — they are the foundation of how communication and learning are organized. They allow instructors to break down conversations into clear, focused spaces where each course, topic, or activity has its own dedicated environment.

Why Channels Matter in Zedu Learning Environments

Channels help to:
- Separate discussions by course, topic, or activity
- Make it easier for students to locate relevant information quickly
- Reduce repeated questions caused by disorganized communication
- Support real-time interaction through Buzz in a structured way
- Improve both teaching delivery and student participation

For example, a university course or bootcamp on Zedu can be organized into channels such as:
- #announcements – where instructors share updates using Buzz
- #week-1-introduction – for onboarding discussions
- #assignments – for submitting and discussing tasks
- #group-projects – for collaboration among students

How to Structure Channels Effectively on Zedu

1. Start with Core Communication Channels
Every workspace should include essential channels such as: Announcements (for official updates), General discussion (for broad conversations), Support (for questions and help).

2. Break Down Learning by Weeks or Modules
For structured programs like bootcamps or courses, create channels based on weekly learning segments or modules/topics. This helps students follow the learning journey step by step.

3. Use Clear and Descriptive Naming
Channel names should immediately tell students what the space is for. Avoid vague names like "discussion1" and instead use "week-2-javascript-basics" or "final-project-guidelines".

4. Balance the Number of Channels
While channels are helpful, too many can become overwhelming. Strike a balance: enough channels to stay organized, but not so many that students feel lost.

The Role of Buzz in Channel Communication

Channels on Zedu are powered by Buzz, which enables real-time messaging, announcements, and interaction. Students can ask questions instantly, instructors can provide updates without delay, and discussions remain active and engaging.

Common Mistakes to Avoid
- Creating too many unnecessary channels
- Using unclear or inconsistent naming
- Mixing unrelated topics in one channel
- Not guiding students on how to use channels

Final Thoughts

Organizing courses with channels on Zedu is not just a technical setup — it is a strategy for delivering better learning experiences. By intentionally structuring channels and leveraging Buzz for communication, instructors can create a learning environment that is organized, engaging, and easy to navigate.`,
		},
		{
			ID:    utility.GenerateUUID(),
			Title: "Running Active Class Discussions on Zedu",
			Content: `Online learning environments can easily become passive if students are not actively engaged. Many students hesitate to participate, while discussions are often dominated by just a few voices. On a platform like Zedu, active discussions are intentionally designed and supported through structured channels and the Buzz communication system.

Why Active Discussions Matter in Zedu Learning Environments

When discussions are active, students are able to:
- Process concepts more deeply through interaction
- Learn from different perspectives shared by their peers
- Build confidence in expressing their thoughts
- Stay engaged throughout the course instead of becoming passive observers

Creating the Right Environment for Participation

1. Create a Safe and Supportive Space
Students are more likely to engage when they feel their opinions are respected. Encourage open expression of ideas, avoid dismissing contributions, and promote respectful interactions.

2. Use Structured Channels for Focused Discussions
Discussions should be organized into dedicated channels based on topics, lessons, or weekly modules. For example: #week-2-discussion, #project-feedback, #exam-preparation.

3. Ask Open-Ended and Thought-Provoking Questions
Instead of asking "Do you understand this topic?", ask "How would you apply this concept in a real-world scenario?" This encourages critical thinking and meaningful responses.

4. Acknowledge and Build on Student Contributions
Simple responses like "That's a great point" or "Interesting perspective" can encourage further participation. Build on responses by asking follow-up questions.

Managing Discussions Effectively on Zedu

Instructors should:
- Keep conversations aligned with learning objectives
- Gently redirect discussions when they go off-topic
- Encourage quieter students to share their thoughts
- Avoid dominating the conversation

The goal is to guide, not control.

The Role of Buzz in Active Discussions

With Buzz, students can ask questions instantly without waiting, instructors can respond quickly and provide clarity, conversations remain active and continuous, and important updates are visible to everyone.

Balancing Participation Across Students

To address participation imbalance, instructors can:
- Direct questions to quieter students
- Encourage peer-to-peer responses
- Create smaller discussion groups within channels
- Set expectations for participation

Common Mistakes to Avoid
- Allowing all discussions to happen in one general channel
- Asking only closed-ended questions
- Ignoring student contributions
- Over-controlling conversations instead of guiding them

Final Thoughts

Running active class discussions on Zedu is about creating an environment where students feel encouraged to participate, share ideas, and learn from one another. By combining structured channels with the real-time capabilities of Buzz, instructors can transform discussions into powerful learning experiences.`,
		},
		{
			ID:    utility.GenerateUUID(),
			Title: "Managing Bootcamp Cohorts on Zedu",
			Content: `Bootcamps are fast-paced, intensive learning environments where students are expected to learn, practice, and deliver results within a short period of time. On Zedu, managing bootcamp cohorts is about creating a structured, centralized system where learning, interaction, and resources are all organized in one place.

Challenges in Bootcamp Learning Environments
- Students often progress at different speeds
- There is a high volume of assignments and deadlines
- Continuous communication is required between students, mentors, and instructors
- Students can easily feel overwhelmed or lost without proper guidance

Structuring Bootcamp Cohorts on Zedu

1. Organize Learning by Weeks or Modules
Create channels based on weekly learning segments or modules/key topics. For example: #week-1-introduction, #week-2-core-concepts, #final-project.

2. Centralize Communication Using Buzz
Buzz ensures that all updates, discussions, and announcements are handled in one place. Students receive real-time updates, instructors can make announcements instantly, and questions and answers remain visible to everyone.

3. Assign Mentors for Support
Mentors can participate in discussions within channels, answer questions through Buzz, and provide feedback on assignments. Having dedicated mentors ensures students receive timely support.

Managing Learning Resources with File Management

Instructors can upload course materials directly into relevant channels, share assignment briefs and resources in one place, and ensure that students always know where to find what they need.

Tracking Progress Across the Cohort

Progress can be monitored through participation in Buzz discussions, completion of tasks within channels, and submission of assignments through shared resources.

Supporting Students Effectively
- Provide timely and constructive feedback
- Encourage collaboration through group channels
- Check in regularly with students who are less active
- Create an environment where students feel comfortable asking questions

Common Mistakes to Avoid
- Overloading students with too much information at once
- Failing to organize channels properly
- Allowing communication to happen across multiple disconnected platforms
- Not providing clear guidance on where to find resources

Final Thoughts

Managing bootcamp cohorts on Zedu is about creating a balance between structure, communication, and support. By leveraging Zedu's tools intentionally, instructors can create a bootcamp experience that is structured, engaging, and capable of delivering strong learning outcomes.`,
		},
		{
			ID:    utility.GenerateUUID(),
			Title: "Onboarding New Students into Zedu",
			Content: `The first experience a student has on a learning platform often determines how engaged, confident, and motivated they will be throughout their learning journey. On Zedu, onboarding is not just about introducing features — it is about guiding students into a structured learning environment where they understand how to navigate, communicate, and access resources from day one.

Understanding the First-Time Student Experience

When a student joins Zedu for the first time, they often face questions like:
- Where do I start?
- How do I find my course or cohort?
- Where are my learning materials?
- How do I communicate with my instructor or classmates?

Why Onboarding Matters in Zedu

Effective onboarding:
- Reduces confusion and frustration
- Helps students become familiar with the platform quickly
- Encourages early engagement and participation
- Builds confidence in navigating learning environments

Key Elements of Effective Onboarding on Zedu

1. Create a Clear "Start Here" Channel
A "Start Here" channel should welcome new students, explain how the platform is structured, and guide students on what to do first.

2. Introduce Buzz for Communication Early
Through Buzz, students can ask questions in real time, instructors can provide updates and announcements, and discussions remain organized within channels.

3. Show Students Where to Find Learning Materials
During onboarding, students should be shown where course materials are uploaded, how to access assignments, and where to find shared resources.

4. Keep Instructions Simple and Human
Use clear, simple language, focus on essential actions first, and introduce features gradually.

5. Provide Immediate Value
Students should experience something useful as quickly as possible — participating in a discussion via Buzz, accessing their first learning material, or completing a simple onboarding task.

Common Mistakes to Avoid
- Showing too many features at once
- Using complex or unclear instructions
- Leaving students to "figure things out" themselves
- Not providing a clear first step

Designing for Confidence, Not Just Completion

A successful onboarding experience ensures that students feel confident navigating the platform, comfortable asking questions through Buzz, clear about where to find resources, and motivated to begin their learning journey.

Final Thoughts

By providing clear guidance, structured communication through Buzz, and easy access to resources through file management, instructors can ensure that students start strong. When students feel supported from the beginning, they are more likely to stay engaged, participate actively, and succeed.`,
		},
		{
			ID:    utility.GenerateUUID(),
			Title: "Tracking Student Progress and Performance on Zedu",
			Content: `In any learning environment, progress is one of the strongest motivators for students. On Zedu, progress tracking is not about monitoring for the sake of control — it is about providing visibility, guidance, and support through structured channels, Buzz interactions, and organized resource management.

Why Progress Tracking Matters in Zedu

Tracking progress helps to:
- Give students a clear sense of direction
- Help instructors identify students who may need additional support
- Encourage accountability and consistency
- Reduce uncertainty about performance

Understanding Different Dimensions of Progress

On Zedu, progress can be viewed through:
- Course completion – finishing modules, lessons, or weekly tasks
- Engagement levels – participation in Buzz discussions and activities
- Performance quality – how well students complete assignments
- Consistency – how regularly students show up and contribute

How Zedu Supports Progress Tracking

1. Structured Channels for Learning Flow
Courses organized into weekly or topic-based channels allow students to follow a clear progression. For example: #week-1-introduction, #week-2-core-concepts, #final-project.

2. Engagement Through Buzz
Instructors can see who is actively contributing to discussions. Participation becomes visible and measurable. Consistent engagement in Buzz often reflects a student's level of involvement in the course.

3. File Submissions and Resource Access
Instructors can monitor assignment submissions, review student work, and provide feedback directly within the platform.

Best Practices for Tracking Progress Effectively

1. Break Learning into Milestones
Large goals can feel overwhelming. Breaking them into smaller milestones helps students stay motivated and focused.

2. Provide Timely and Actionable Feedback
Feedback should be clear, constructive, and given promptly.

3. Encourage Self-Reflection
Students should be encouraged to assess their own progress: What have I learned so far? What challenges am I facing? What should I focus on next?

4. Maintain Consistency in Tracking
Progress tracking should be consistent throughout the course. Regular check-ins and updates help students stay aligned and aware of their progress.

Common Mistakes to Avoid
- Focusing only on completion rather than understanding
- Providing delayed or unclear feedback
- Ignoring student engagement levels
- Overcomplicating tracking systems

Final Thoughts

Tracking student progress on Zedu transforms learning from guesswork into a structured and guided journey. By combining structured channels, active participation through Buzz, and organized file management, Zedu creates a system where progress is visible, measurable, and meaningful.`,
		},
		{
			ID:    utility.GenerateUUID(),
			Title: "Building a Strong Learning Community on Zedu",
			Content: `Learning is not just an individual activity — it thrives in a community. When students feel connected to others, they are more likely to participate, stay motivated, and successfully complete their learning journey. Zedu addresses this by combining structured channels with real-time communication through Buzz, creating an environment where interaction is natural, consistent, and meaningful.

Why Community Matters in Zedu Learning Environments

A strong learning community helps to:
- Reduce feelings of isolation in online learning
- Increase participation and engagement
- Encourage collaboration and peer learning
- Improve motivation and course completion rates

The Role of Buzz in Building Community

Through Buzz, students can ask questions and receive immediate responses, conversations remain active and engaging, instructors can connect with students beyond formal teaching, and peer-to-peer learning becomes more accessible.

Key Elements of a Strong Learning Community

1. Open and Accessible Communication
On Zedu, this is achieved through general discussion channels, topic-based channels, and support channels.

2. Psychological Safety
Students must feel safe expressing their thoughts without fear of judgment. Instructors can encourage this by respecting all contributions, promoting inclusive discussions, and setting clear expectations for respectful communication.

3. Shared Goals and Learning Purpose
A strong community is built around shared goals — completing a project together, preparing for an assessment, or solving problems as a group.

4. Recognition and Encouragement
Acknowledging student contributions — highlighting thoughtful responses, recognizing progress, encouraging participation — can have a significant impact on engagement.

Encouraging Meaningful Interaction

Create discussion opportunities that go beyond surface-level interaction. Promote peer learning by encouraging students to answer each other's questions. Support informal conversations to help students build relationships.

Sustaining the Community Over Time
- Keep discussions active through Buzz
- Introduce regular activities or prompts
- Monitor and guide conversations
- Ensure that no student feels left out

Common Mistakes to Avoid
- Relying only on formal communication
- Ignoring inactive students
- Allowing discussions to become disorganized
- Failing to create a welcoming environment

Final Thoughts

Building a strong learning community on Zedu transforms education from an individual task into a shared experience. By leveraging structured channels and real-time communication through Buzz, instructors can create an environment where students collaborate, participate, and grow together.`,
		},
		{
			ID:    utility.GenerateUUID(),
			Title: "Managing Classroom Communication with Buzz on Zedu",
			Content: `Effective communication is at the heart of any successful learning environment. Zedu addresses the challenge of scattered communication through Buzz — its built-in communication system that enables real-time, structured interaction within learning environments. Buzz is not just a messaging feature; it is the backbone of how communication happens on Zedu.

Why Communication Matters in Zedu Learning Environments

Communication plays a critical role in learning because it allows students to:
- Clarify concepts they do not fully understand
- Engage in discussions with peers
- Receive timely updates from instructors
- Stay connected to the learning process

Understanding Buzz as a Communication Tool

Unlike traditional messaging systems where conversations become cluttered, Buzz ensures that communication is:
- Organized within specific channels
- Easy to follow and revisit
- Accessible to all relevant participants
- Integrated directly into the learning workflow

Key Features of Buzz on Zedu

1. Real-Time Messaging – Questions can be asked and answered without delay, reducing confusion and improving understanding.

2. Structured Communication Through Channels – Conversations are grouped based on topics, lessons, or activities, ensuring discussions remain relevant and organized.

3. Visibility and Transparency – Messages are visible to everyone in the channel, allowing students to learn from each other's questions and responses.

4. Continuous Engagement – Buzz keeps conversations active beyond scheduled sessions, allowing students to stay engaged even outside formal class time.

Best Practices for Managing Communication with Buzz

1. Use Dedicated Channels for Different Topics
Organize discussions into separate channels: #announcements, #general-discussion, #assignment-questions.

2. Keep Messages Clear and Purposeful
Messages should be easy to understand and directly related to the topic being discussed. Avoid long, unclear messages or off-topic discussions in focused channels.

3. Encourage Student Participation
Instructors can prompt discussions, ask follow-up questions, and recognize contributions to create a more interactive environment.

4. Maintain Consistency in Communication
Regular updates and consistent engagement help students stay connected. Share updates frequently, respond to questions promptly, and keep communication active.

Common Mistakes to Avoid
- Using a single channel for all communication
- Ignoring student questions or delays in responses
- Allowing discussions to become disorganized
- Overloading students with too many messages

Final Thoughts

Managing classroom communication with Buzz on Zedu is about creating a structured, interactive, and engaging communication system. By using Buzz within well-organized channels, instructors can ensure that students remain informed, connected, and actively involved in their learning journey.`,
		},
		{
			ID:    utility.GenerateUUID(),
			Title: "Managing Files and Learning Resources on Zedu",
			Content: `Access to learning materials is a critical part of any educational experience. Zedu solves the problem of scattered files through its integrated file management system, which ensures that all learning resources are stored, organized, and easily accessible within the platform.

Why File Management Matters in Zedu Learning Environments

Effective file management helps to:
- Ensure that students can easily access learning materials at any time
- Reduce confusion caused by scattered or missing files
- Improve organization across courses, topics, and assignments
- Support a smoother and more efficient learning process

How Zedu Organizes Learning Resources

Zedu integrates file management directly into its structured channel system. Instead of uploading files randomly, instructors can:
- Attach materials to specific channels
- Share resources within relevant discussions
- Keep files aligned with topics or modules

For example:
- Files related to Week 1 can be shared in #week-1-introduction
- Assignment documents can be uploaded in #assignments
- Project resources can be placed in #group-projects

The Role of File Management in Supporting Learning

1. Centralized Access to Resources
All learning materials are available within the platform. Students can access notes, assignments, and resources in one place, revisit materials whenever needed, and stay organized without confusion.

2. Contextual Learning
Because files are shared within channels, students understand the context of each resource. A file shared in a "week-2" channel is clearly linked to that topic.

3. Improved Collaboration
File sharing on Zedu supports collaboration among students — sharing resources with teammates, working together on projects, and accessing shared documents easily.

Best Practices for Managing Files on Zedu

1. Upload Files in the Right Channels – Files should always be uploaded in channels that match their purpose.
2. Use Clear and Descriptive File Names – Use "week-2-javascript-notes.pdf" instead of "document1.pdf".
3. Keep Resources Organized and Updated – Outdated or irrelevant files should be removed or updated.
4. Avoid Overloading Channels with Files – Keep only relevant materials and avoid unnecessary duplication.

The Connection Between File Management and Communication

Instructors can share a file and immediately discuss it through Buzz. Students can ask questions about a resource in the same channel. Feedback on assignments can be given alongside the submitted files.

Common Mistakes to Avoid
- Sharing files without proper organization
- Using unclear file names
- Uploading resources in unrelated channels
- Relying on external platforms instead of centralizing files

Final Thoughts

Managing files and learning resources on Zedu is about creating a structured, accessible, and efficient learning environment. By using Zedu's file management system alongside Buzz communication, instructors can create a seamless learning experience where everything students need is available, organized, and easy to use.`,
		},
		{
			ID:    utility.GenerateUUID(),
			Title: "Using Live Classes and Collaboration Tools on Zedu",
			Content: `Learning becomes more effective when students are actively participating in real-time interactions. Zedu addresses the challenge of disconnected live teaching by integrating live classes and collaboration tools directly into its platform, ensuring that teaching, communication, and interaction all happen in one structured environment.

Why Live Classes and Collaboration Matter in Zedu

Live interaction helps to:
- Provide immediate clarification of concepts
- Encourage active participation during lessons
- Create opportunities for real-time collaboration
- Strengthen the connection between students and instructors

How Zedu Supports Live Classes

Through live classes, instructors can:
- Deliver lessons in real time
- Explain concepts with immediate interaction
- Answer questions as they arise
- Engage students through discussions

Collaboration Tools for Interactive Learning

Zedu supports collaboration through:
- Shared discussions within channels
- Real-time communication through Buzz
- Interactive tools such as whiteboards

These tools enable students to share ideas and perspectives, work on group projects, and collaborate on problem-solving tasks.

The Role of Buzz in Live Learning

During and after live sessions:
- Students can ask questions instantly
- Instructors can share updates or clarifications
- Discussions can continue beyond the live session

This ensures that learning does not stop when the session ends, but continues through ongoing interaction.

Best Practices for Using Live Classes on Zedu

1. Prepare Before the Session – Plan the structure, share necessary materials in advance, and set clear objectives.
2. Encourage Active Participation – Ask questions during the session, invite students to share ideas, and use interactive tools like whiteboards.
3. Use Channels to Support Sessions – Share session materials, continue discussions after the class, and provide additional resources.
4. Follow Up After Sessions – Summarize key points, share additional resources, and address unanswered questions through Buzz.

Common Challenges and How to Address Them
- Low student participation → Encourage interaction during sessions
- Technical difficulties → Use structured channels for follow-ups
- Disorganized follow-up discussions → Leverage Buzz for ongoing communication

Final Thoughts

Using live classes and collaboration tools on Zedu transforms learning from passive consumption into active participation. When live learning is combined with continuous collaboration, the overall learning experience becomes more dynamic, connected, and impactful.`,
		},
		{
			ID:    utility.GenerateUUID(),
			Title: "Managing Roles and Permissions on Zedu",
			Content: `In any structured learning environment, not every participant has the same responsibilities or level of access. Zedu solves this through its role-based access system, which ensures that every participant has the appropriate level of access based on their responsibilities.

Why Roles and Permissions Matter in Zedu

Managing roles and permissions helps to:
- Ensure that students only access relevant learning materials
- Allow instructors to manage content and communication effectively
- Prevent unauthorized changes to important resources
- Maintain order and accountability within the workspace

Understanding Roles on Zedu

Zedu typically includes different types of roles:
- Instructors – responsible for managing courses, creating content, and guiding students
- Mentors – provide support, answer questions, and assist students during learning
- Students – participate in learning activities, discussions, and assignments

How Permissions Support Structured Learning

- Instructors can create and manage channels
- Mentors can support discussions and provide guidance
- Students can participate in discussions and access resources

The Connection Between Roles, Communication, and Resources

Through Buzz: Instructors can make announcements, mentors can respond to student questions, and students can participate in discussions. Through file management: Instructors upload and manage learning materials, students access resources without modifying them, and sensitive documents remain protected.

Best Practices for Managing Roles and Permissions

1. Assign Roles Clearly from the Start – Before a course or bootcamp begins, all participants should be assigned their roles.
2. Limit Access to What Is Necessary – Students should not be able to modify core course content; only instructors should manage major structural changes.
3. Keep Permissions Consistent Across the Platform – Consistency helps users understand how the platform works.
4. Review and Update Roles When Needed – As courses progress, roles may need to be adjusted.

Common Mistakes to Avoid
- Giving too much access to all participants
- Not clearly defining responsibilities
- Allowing students to modify important content
- Failing to update roles when changes occur

Final Thoughts

Managing roles and permissions on Zedu is not just about restricting access — it is about creating a structured, secure, and efficient learning environment. When each participant has clearly defined responsibilities and appropriate access, communication flows better, resources are managed effectively, and the overall learning experience becomes more organized.`,
		},
	}

	for _, guide := range guides {
		var existing models.ZeduGuide
		if err := db.Unscoped().Where("title = ?", guide.Title).First(&existing).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				if err := db.Create(&guide).Error; err != nil {
					logger.Error("failed to create zedu guide: " + err.Error())
				}
			} else {
				logger.Error("failed to query zedu guide: " + err.Error())
			}
		} else {
			guide.ID = existing.ID
			guide.DeletedAt = gorm.DeletedAt{Valid: false}
			if err := db.Unscoped().Model(&existing).Updates(guide).Error; err != nil {
				logger.Error("failed to update zedu guide: " + err.Error())
			}
		}
		logger.Info("Zedu guides seeded successfully>>>>>>>>>>>>>>")
	}
}
