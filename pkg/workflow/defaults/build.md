Build tailored application materials for the job posting.

## Workflow

1. **Read all reference materials:**

   - Read all files in `{{.ReferencePath}}` - skills inventory, guidelines, boundaries
   - Read `{{.CVPath}}` - current CV content

2. **Analyze job requirements:**

   - Identify required vs nice-to-have skills
   - Match experiences from reference to requirements
   - Note gaps honestly

3. **Tailor documents:**

   - CV: emphasize relevant sections per role type
   - Cover letter: follow structure from reference, use ONLY verified experiences
   - Identify keywords to incorporate

4. **Write files:**

   - Update the CV file with tailored content
   - Write the cover letter to the appropriate location
   - Ensure all changes are saved to disk

5. **Verify accuracy:**

   - List every claim and cite source from CV or reference files
   - Never fabricate or exaggerate experiences

6. **Build & preview:**

   - If makefile exists: `make combined && code build/combined.pdf`

7. **Apply feedback iterations:**
   - When user provides feedback via `-c` flag, apply changes
   - Rebuild and reopen preview
   - Repeat until user is satisfied

## Critical Rules

- Use ONLY experiences documented in CV or reference files
- Follow guidelines from reference directory exactly
- Never fabricate or exaggerate experiences
- Cite sources for every claim made
