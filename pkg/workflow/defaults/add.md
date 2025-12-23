Fetch job posting details from the provided URL and extract structured information.

## Workflow

1. **Get job details**:

   - If user provided full job text: Use that directly (skip WebFetch)
   - If only URL provided:
     - Use WebFetch with prompt that requests:
       - Extract metadata (title, company, location, etc.)
       - Return the COMPLETE, VERBATIM, UNEDITED full job posting text
     - If WebFetch fails: Inform user to provide the job text directly with --text flag

2. **Extract key information**:

   - Job title (use only the job title, not company name)
   - Company name
   - Location (remote/city/hybrid)
   - Department/team (if available)
   - Hiring manager, title, contact info (if mentioned)
   - Deadline (if mentioned)
   - Full job ad text (EXACT original text, not summary)

3. **Return structured JSON output**:

   ```json
   {
     "url": "[URL]",
     "title": "[Job Title]",
     "company": "[Company]",
     "location": "[Location or null]",
     "department": "[Department/Team or null]",
     "hiring_manager": "[Hiring Manager or null]",
     "deadline": "[YYYY-MM-DD format if found, or null]",
     "full_job_ad": "[COMPLETE VERBATIM job posting text]"
   }
   ```

   **Field Rules**:
   - Use `null` for fields that are not found/specified
   - `deadline`: If mentioned, use YYYY-MM-DD format. If not mentioned, use `null`

   **CRITICAL**: The `full_job_ad` field must contain the EXACT, COMPLETE, UNEDITED text from the job posting. Do NOT summarize, paraphrase, or rewrite.

## Important

- Extract exactly the fields defined in the schema
- Use proper JSON format with null for missing values
- Return only valid JSON, no markdown code blocks
