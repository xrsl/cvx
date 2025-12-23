Tailor application materials for the job posting.

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

4. **Verify accuracy:**
   - List every claim and cite source from CV or reference files
   - Never fabricate or exaggerate experiences

5. **Build & preview:**
   - If makefile exists: `make combined && code build/combined.pdf`

6. **Wait for user review**

7. **Apply feedback iterations:**
   - When user provides feedback, apply changes
   - Rebuild and reopen preview
   - Repeat until user says "approved" or "looks good"

## After Approval

When user says "approved", "looks good", "submit", or similar:

1. **Commit changes:** Format as "Tailored application for [Company] [Role]"
2. **Create tag:** `git tag <issue>-<company>-<role>-YYYY-MM-DD && git push origin <tag>`
3. **Update project status to "Applied" and set AppliedDate to today:**
   ```bash
   # Get project item ID for the issue
   gh api graphql -f query='{ repository(owner:"OWNER", name:"REPO") { issue(number:ISSUE) { projectItems(first:1) { nodes { id } } } } }'
   # Update status field to "Applied" and AppliedDate to today's date
   ```

## Critical Rules

- Use ONLY experiences documented in CV or reference files
- Follow guidelines from reference directory exactly
- Never fabricate or exaggerate experiences
- Cite sources for every claim made
