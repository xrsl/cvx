┌────────────┐
│ User │
│ cvx build │
└─────┬──────┘
│
▼
┌───────────────┐
│ Go CLI (cvx) │
│ - loads CV │
│ - loads job │
│ - computes │
│ cache hash │
└─────┬─────────┘
│ cache miss
▼
┌────────────────────────┐
│ uv tool run cvx-agent │
│ (isolated Python env) │
└─────┬──────────────────┘
│ structured JSON
▼
┌───────────────┐
│ schema.json │
│ validation │
└─────┬─────────┘
▼
┌───────────────┐
│ cv.yaml │
│ letter.yaml │
└───────────────┘
