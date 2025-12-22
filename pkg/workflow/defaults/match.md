Analyze job match quality for the provided position.

## Workflow

1. **Get job posting content:**
   - Use the provided job text or URL content

2. **Read candidate's profile:**
   - Read `{{.ExperiencePath}}` - skills inventory and gaps
   - Read `{{.CVPath}}` - job titles, dates, responsibilities

3. **Analyze match across:**
   - Experience years required vs actual
   - Technical skills alignment (highlight gaps)
   - Domain knowledge fit
   - Role type alignment
   - Career direction fit

4. **Provide honest assessment:**
   - Rate match: Strong/Moderate/Weak/Poor
   - Strengths with ✓
   - Gaps with ❌ (cite specific missing skills)
   - Clear recommendation with reasoning

**Be direct about gaps. Reference skills inventory explicitly when noting missing skills.**
