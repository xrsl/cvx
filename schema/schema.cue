package cv

// ============================================================================
// Utility Types
// ============================================================================

#Date:      string | int | null
#ExactDate: string | int | "present" | null

#SocialNetwork: {
	network:  "LinkedIn" | "GitHub" | "GitLab" | "Instagram" | "Twitter" | "X" | string
	username: string
}

// ============================================================================
// Entry Types
// ============================================================================

// Experience & Education entry (shares same structure in this project)
#ExperienceEntry: {
	company?:    string | null
	position?:   string | null
	location?:   string | null
	start_date?: #Date
	end_date?:   #ExactDate
	date?:       #Date
	summary?:    string | null
	highlights?: [...string] | null
	...
}

// Education entry (supports both custom company/position style and standard institution/area/degree style)
#EducationEntry: {
	company?:     string | null
	position?:    string | null
	institution?: string | null
	area?:        string | null
	degree?:      string | null
	location?:    string | null
	start_date?:  #Date
	end_date?:    #ExactDate
	date?:        #Date
	summary?:     string | null
	highlights?: [...string] | null
	...
}

// Publication entry
#PublicationEntry: {
	title: string | null
	authors?: [...string] | null
	summary?: string | null
	doi?:     string | null
	url?:     string | null
	journal?: string | null
	date?:    #Date
	...
}

// One-line entry (e.g. for skills)
#OneLineEntry: {
	label:   string
	details: string
	...
}

// Simple bullet entry
#BulletEntry: {
	bullet: string
	...
}

// ============================================================================
// CV Schema
// ============================================================================

#CV: {
	name?:     string | null
	headline?: string | null
	location?: string | null
	email?: string | [...string] | null
	phone?: string | [...string] | null
	website?: string | [...string] | null
	social_networks?: [...#SocialNetwork] | null
	expertise_tags?: [...string] | null
	sections?: {
		summary?:    string | null
		motivation?: string | null
		experience?: [...#ExperienceEntry] | null
		education?: [...#EducationEntry] | null
		publications?: [...#PublicationEntry] | null
		skills?: [...#OneLineEntry] | null
		honors_and_awards?: [...#ExperienceEntry] | null
		courses?: [...#ExperienceEntry] | null
		values?: [...#BulletEntry] | null
		hobbies?: [...#BulletEntry] | null
		references?: [...string] | null
		[string]: [...] | string | null
	}
	...
}

// Main CV wrapper
#CVSchema: {
	cv: #CV
}

// ============================================================================
// Letter Schema Definitions
// ============================================================================

#Sender: {
	name:      string
	phone?:    string | null
	email:     string
	linkedin?: string | null
	github?:   string | null
	address?:  string | null
}

#Recipient: {
	name:    string
	title?:  string | null
	company: string
	address: string
}

#BodyParagraph: {
	paragraph: string
}

#Content: {
	salutation: string
	opening:    string
	body: [...#BodyParagraph]
	closing: string
}

#Metadata: {
	date:              string | "auto"
	position_applied?: string | null
}

#Letter: {
	sender:    #Sender
	recipient: #Recipient
	content:   #Content
	metadata:  #Metadata
}

// Letter schema (for validation)
#LetterSchema: {
	letter: #Letter
}

// ============================================================================
// Unified Schema (for JSON export)
// ============================================================================

#UnifiedSchema: {
	cv?:     #CV
	letter?: #Letter
}
