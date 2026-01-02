# Schema Reference

When using Python agent mode (`cvx build -m <model>`), CV and letter data are stored in YAML files that conform to `schema/schema.json`.

## Overview

**Files:**

- `src/cv.yaml` - Structured CV data
- `src/letter.yaml` - Structured cover letter data
- `schema/schema.json` - JSON Schema definition

**Format:**

```yaml
cv:
  name: "Your Name"
  email: "you@example.com"
  # ... more fields

letter:
  sender:
    name: "Your Name"
    email: "you@example.com"
  recipient:
    name: "Hiring Manager"
    company: "Company Name"
  # ... more fields
```

## CV Schema

### Top-Level Fields

```yaml
cv:
  name: "John Doe" # Full name
  email: "john@example.com" # Email address (string or array)
  phone: "+1-234-567-8900" # Phone number (string or array)
  location: "San Francisco, CA" # Current location
  headline: "Senior Data Scientist" # Professional headline
  website: "https://johndoe.com" # Personal website (string or array)

  expertise_tags: # List of key skills/technologies
    - "Machine Learning"
    - "Python"
    - "TensorFlow"

  social_networks: # Social media profiles
    - network: "LinkedIn"
      username: "johndoe"
    - network: "GitHub"
      username: "jdoe"
```

### Sections

The `sections` field contains all CV content:

#### Experience

```yaml
cv:
  sections:
    experience:
      - company: "Tech Corp"
        position: "Senior Data Scientist"
        location: "San Francisco, CA"
        start_date: "2020-01"
        end_date: "present"
        summary: "Led ML initiatives across the organization"
        highlights:
          - "Improved model accuracy by 25%"
          - "Built scalable ML pipeline handling 1M+ requests/day"
          - "Mentored team of 5 junior data scientists"
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

```yaml
cv:
  sections:
    education:
      - institution: "University of California"
        degree: "Ph.D."
        area: "Computer Science"
        location: "Berkeley, CA"
        start_date: 2015
        end_date: 2019
        summary: "Focus on machine learning and neural networks"
        highlights:
          - "Published 5 papers in top-tier conferences"
          - "GPA: 3.9/4.0"
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

```yaml
cv:
  sections:
    skills:
      - label: "Programming"
        details: "Python, R, SQL, Java, Go"
      - label: "ML/AI"
        details: "TensorFlow, PyTorch, Scikit-learn, XGBoost"
      - label: "Cloud"
        details: "AWS (SageMaker, Lambda, S3), GCP, Azure"
```

**Fields:**

- `label` (string, required) - Skill category
- `details` (string, required) - Comma-separated or descriptive text

#### Publications

```yaml
cv:
  sections:
    publications:
      - title: "Deep Learning for Time Series Forecasting"
        authors:
          - "John Doe"
          - "Jane Smith"
        journal: "Journal of Machine Learning Research"
        date: 2023
        doi: "10.1234/jmlr.2023.1234"
        url: "https://jmlr.org/papers/v24/doe23.html"
        summary: "Novel architecture for multivariate time series"
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

```yaml
cv:
  sections:
    # Professional summary
    summary: "Experienced data scientist with 8+ years..."

    # Motivation statement
    motivation: "Passionate about using ML to solve real-world problems..."

    # Honors and awards
    honors_and_awards:
      - company: "Tech Corp"
        position: "Employee of the Year"
        date: 2023
        summary: "Recognized for outstanding contributions"

    # Courses/certifications
    courses:
      - company: "Coursera"
        position: "Deep Learning Specialization"
        date: 2020
        summary: "5-course specialization by Andrew Ng"

    # Hobbies/interests
    hobbies:
      - bullet: "Marathon running"
      - bullet: "Open source contributions"

    # Personal values
    values:
      - bullet: "Continuous learning and growth"
      - bullet: "Collaboration and mentorship"

    # References
    references:
      - "Available upon request"
```

## Letter Schema

### Structure

```yaml
letter:
  sender:
    name: "John Doe"
    email: "john@example.com"
    phone: "+1-234-567-8900"
    address: "123 Main St, San Francisco, CA 94102"
    linkedin: "johndoe"
    github: "jdoe"

  recipient:
    name: "Jane Smith"
    title: "Hiring Manager"
    company: "Tech Corp"
    address: "456 Tech Blvd, San Francisco, CA 94103"

  metadata:
    date: "2025-01-15" # or "auto" for automatic date
    position_applied: "Senior Data Scientist"

  content:
    salutation: "Dear Ms. Smith"

    opening: "I am writing to express my strong interest in the Senior Data Scientist position at Tech Corp."

    body:
      - paragraph: "With over 8 years of experience in machine learning and data science, I have developed a deep expertise in building scalable ML systems. At my current role at AI Startup, I led the development of a recommendation engine that increased user engagement by 40%."

      - paragraph: "What excites me most about this opportunity is Tech Corp's commitment to using AI for social good. Your recent work on healthcare AI aligns perfectly with my passion for applying ML to solve real-world problems."

      - paragraph: "I am particularly impressed by your team's publication on federated learning, and I would be thrilled to contribute to such cutting-edge research while helping drive business impact."

    closing: "Thank you for considering my application. I look forward to the opportunity to discuss how my experience and passion align with Tech Corp's mission."
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

### Body Paragraphs

```yaml
letter:
  content:
    body:
      - paragraph: "First body paragraph text..."
      - paragraph: "Second body paragraph text..."
      - paragraph: "Third body paragraph text..."
```

Each paragraph is an object with a single `paragraph` field.

## Data Types

### Date

Flexible date format:

```yaml
date: "2023-01"      # String: YYYY-MM
date: 2023           # Integer: year only
date: null           # No date
```

### ExactDate

Date with "present" support:

```yaml
end_date: "2023-12"  # String: YYYY-MM
end_date: 2023       # Integer: year only
end_date: "present"  # Literal: current/ongoing
end_date: null       # No date
```

### Arrays vs. Strings

Some fields accept both single values and arrays:

```yaml
email: "john@example.com"              # Single
email:                                 # Multiple
  - "john@example.com"
  - "jdoe@company.com"

phone: "+1-234-567-8900"               # Single
phone:                                 # Multiple
  - "+1-234-567-8900"
  - "+1-987-654-3210"

website: "https://johndoe.com"         # Single
website:                               # Multiple
  - "https://johndoe.com"
  - "https://blog.johndoe.com"
```

## Example Files

### Minimal cv.yaml

```yaml
cv:
  name: "John Doe"
  email: "john@example.com"
  phone: "+1-234-567-8900"
  location: "San Francisco, CA"

  sections:
    experience:
      - company: "Tech Corp"
        position: "Data Scientist"
        start_date: "2020"
        end_date: "present"
        highlights:
          - "Built ML models"
          - "Improved accuracy by 20%"

    education:
      - institution: "University"
        degree: "M.S."
        area: "Computer Science"
        end_date: 2020

    skills:
      - label: "Programming"
        details: "Python, SQL, R"
```

### Minimal letter.yaml

```yaml
letter:
  sender:
    name: "John Doe"
    email: "john@example.com"

  recipient:
    name: "Hiring Manager"
    company: "Tech Corp"
    address: "123 Tech Blvd, City, ST 12345"

  metadata:
    date: "auto"

  content:
    salutation: "Dear Hiring Manager"
    opening: "I am writing to apply for the Data Scientist position."
    closing: "Thank you for your consideration."
```

## Validation

The Python agent automatically validates all output against `schema/schema.json` using Pydantic models:

- **Type checking**: Ensures all fields have correct types
- **Required fields**: Validates all required fields are present
- **Schema conformance**: Rejects invalid structures
- **Additional properties**: Some objects allow extra fields for flexibility

If validation fails, the Python agent will retry or fall back to an alternative model.

## Generating schema.json

The Pydantic models in `agent/cvx_agent/models.py` are auto-generated from `schema/schema.json` using:

```bash
datamodel-codegen --input schema/schema.json --output agent/cvx_agent/models.py
```

This ensures the Python agent always validates against the canonical schema.
