# Schema Reference

When using agent mode (`cvx build -m <model>`), CV and letter data are stored in TOML files that conform to `schema/schema.json`.

## Overview

**Files:**

- `src/cv.toml` - Structured CV data
- `src/letter.toml` - Structured cover letter data
- `schema/schema.json` - JSON Schema definition

**Validation:**

The agent uses [pydantic-ai](https://ai.pydantic.dev/) for structured output. Models are auto-generated from the JSON Schema using `datamodel-codegen`.

## CV Schema

### Top-Level Fields

```toml
[cv]
name = "John Doe"
email = "john@example.com"
phone = "+1-234-567-8900"
location = "San Francisco, CA"
headline = "Senior Data Scientist"
website = "https://johndoe.com"

expertise_tags = [
    "Machine Learning",
    "Python",
    "TensorFlow"
]

[[cv.social_networks]]
network = "LinkedIn"
username = "johndoe"

[[cv.social_networks]]
network = "GitHub"
username = "jdoe"
```

### Sections

The `sections` field contains all CV content:

#### Experience

```toml
[[cv.sections.experience]]
company = "Tech Corp"
position = "Senior Data Scientist"
location = "San Francisco, CA"
start_date = "2020-01"
end_date = "present"
summary = "Led ML initiatives across the organization"
highlights = [
    "Improved model accuracy by 25%",
    "Built scalable ML pipeline handling 1M+ requests/day",
    "Mentored team of 5 junior data scientists"
]
```

**Fields:**

- `company` (string) - Company name
- `position` (string) - Job title
- `location` (string) - Office location
- `start_date` (string/int) - Start date (YYYY-MM or YYYY)
- `end_date` (string/int/"present") - End date or "present"
- `date` (string/int) - Alternative single date field
- `summary` (string) - Brief role description
- `highlights` (array) - Key achievements and responsibilities

#### Education

```toml
[[cv.sections.education]]
institution = "University of California"
degree = "Ph.D."
area = "Computer Science"
location = "Berkeley, CA"
start_date = 2015
end_date = 2019
summary = "Focus on machine learning and neural networks"
highlights = [
    "Published 5 papers in top-tier conferences",
    "GPA: 3.9/4.0"
]
```

**Fields:**

- `institution` (string) - University/school name
- `degree` (string) - Degree type (B.S., M.S., Ph.D., etc.)
- `area` (string) - Field of study
- `location` (string) - Campus location
- `start_date` (string/int) - Start year
- `end_date` (string/int) - End year
- `summary` (string) - Description
- `highlights` (array) - Achievements

#### Skills

```toml
[[cv.sections.skills]]
label = "Programming"
details = "Python, R, SQL, Java, Go"

[[cv.sections.skills]]
label = "ML/AI"
details = "TensorFlow, PyTorch, Scikit-learn, XGBoost"

[[cv.sections.skills]]
label = "Cloud"
details = "AWS (SageMaker, Lambda, S3), GCP, Azure"
```

**Fields:**

- `label` (string, required) - Skill category
- `details` (string, required) - Comma-separated or descriptive text

#### Publications

```toml
[[cv.sections.publications]]
title = "Deep Learning for Time Series Forecasting"
authors = ["John Doe", "Jane Smith"]
journal = "Journal of Machine Learning Research"
date = 2023
doi = "10.1234/jmlr.2023.1234"
url = "https://jmlr.org/papers/v24/doe23.html"
summary = "Novel architecture for multivariate time series"
```

**Fields:**

- `title` (string, required) - Paper title
- `authors` (array) - List of authors
- `journal` (string) - Publication venue
- `date` (string/int) - Publication year
- `doi` (string) - DOI identifier
- `url` (string) - Paper URL
- `summary` (string) - Brief description

#### Other Sections

```toml
[cv.sections]
summary = "Experienced data scientist with 8+ years..."
motivation = "Passionate about using ML to solve real-world problems..."

[[cv.sections.honors_and_awards]]
company = "Tech Corp"
position = "Employee of the Year"
date = 2023
summary = "Recognized for outstanding contributions"

[[cv.sections.courses]]
company = "Coursera"
position = "Deep Learning Specialization"
date = 2020
summary = "5-course specialization by Andrew Ng"

[[cv.sections.hobbies]]
bullet = "Marathon running"

[[cv.sections.hobbies]]
bullet = "Open source contributions"

[[cv.sections.values]]
bullet = "Continuous learning and growth"

[[cv.sections.values]]
bullet = "Collaboration and mentorship"

cv.sections.references = ["Available upon request"]
```

## Letter Schema

### Structure

```toml
[letter.sender]
name = "John Doe"
email = "john@example.com"
phone = "+1-234-567-8900"
address = "123 Main St, San Francisco, CA 94102"
linkedin = "johndoe"
github = "jdoe"

[letter.recipient]
name = "Jane Smith"
title = "Hiring Manager"
company = "Tech Corp"
address = "456 Tech Blvd, San Francisco, CA 94103"

[letter.metadata]
date = "2025-01-15"  # or "auto" for automatic date
position_applied = "Senior Data Scientist"

[letter.content]
salutation = "Dear Ms. Smith"
opening = "I am writing to express my strong interest..."
closing = "Thank you for considering my application..."

[[letter.content.body]]
paragraph = "With over 8 years of experience..."

[[letter.content.body]]
paragraph = "What excites me most about this opportunity..."

[[letter.content.body]]
paragraph = "I am particularly impressed by your team's work..."
```

### Required Fields

**Sender:**

- `name` (required)
- `email` (required)
- `phone`, `address`, `linkedin`, `github` (optional)

**Recipient:**

- `name` (required)
- `company` (required)
- `address` (required)
- `title` (optional)

**Metadata:**

- `date` (required) - Specific date (YYYY-MM-DD) or "auto"
- `position_applied` (optional)

**Content:**

- `salutation` (required) - Opening greeting
- `opening` (required) - First paragraph
- `body` (optional) - Array of body paragraphs
- `closing` (required) - Final paragraph

## Data Types

### Date

Flexible date format:

```toml
date = "2023-01"      # String: YYYY-MM
date = 2023           # Integer: year only
# date omitted = no date
```

### ExactDate

Date with "present" support:

```toml
end_date = "2023-12"  # String: YYYY-MM
end_date = 2023       # Integer: year only
end_date = "present"  # Literal: current/ongoing
# end_date omitted = no date
```

### Arrays

```toml
# Multiple emails
email = ["john@example.com", "jdoe@company.com"]

# Multiple phones
phone = ["+1-234-567-8900", "+1-987-654-3210"]

# Multiple websites
website = ["https://johndoe.com", "https://blog.johndoe.com"]
```

## Example Files

### Minimal cv.toml

```toml
#:schema ../schema/schema.json

[cv]
name = "John Doe"
email = "john@example.com"
phone = "+1-234-567-8900"
location = "San Francisco, CA"

[[cv.sections.experience]]
company = "Tech Corp"
position = "Data Scientist"
start_date = "2020"
end_date = "present"
highlights = [
    "Built ML models",
    "Improved accuracy by 20%"
]

[[cv.sections.education]]
institution = "University"
degree = "M.S."
area = "Computer Science"
end_date = 2020

[[cv.sections.skills]]
label = "Programming"
details = "Python, SQL, R"
```

### Minimal letter.toml

```toml
#:schema ../schema/schema.json

[letter.sender]
name = "John Doe"
email = "john@example.com"

[letter.recipient]
name = "Hiring Manager"
company = "Tech Corp"
address = "123 Tech Blvd, City, ST 12345"

[letter.metadata]
date = "auto"

[letter.content]
salutation = "Dear Hiring Manager"
opening = "I am writing to apply for the Data Scientist position."
closing = "Thank you for your consideration."
```

## Validation

The agent validates all output using Pydantic models generated from `schema/schema.json`:

- **Type checking**: Ensures all fields have correct types
- **Required fields**: Validates all required fields are present
- **Schema conformance**: Rejects invalid structures
- **Automatic retry**: Retries or falls back to alternative model on failure

## IDE Support

TOML files support IDE autocompletion via schema references:

```toml
#:schema ../schema/schema.json

[cv]
name = "..."
```

Schema comments are automatically added by `cvx build`.

## Generating Pydantic Models

Models are auto-generated from the schema:

```bash
datamodel-codegen --input schema/schema.json --output agent/cvx_agent/models.py
```

The Go CLI automatically regenerates models when the schema changes.
