---
title: Stoicism
tags: [philosophy, stoicism, ancient-greece]
author: Various
status: evergreen
source: https://en.wikipedia.org/wiki/Stoicism
---

# Stoicism

Stoicism is a school of Hellenistic philosophy founded by [[Zeno of Citium]] in Athens in the early 3rd century BC. It is a philosophy of personal virtue ethics informed by its system of logic and its views on the natural world.

## Core Doctrines

According to the Stoics, the path to *eudaimonia* (happiness, or blessedness) is found in accepting the moment as it presents itself, by not allowing oneself to be controlled by the desire for pleasure or fear of pain.

> "You have power over your mind, not outside events. Realize this, and you will find strength." — Marcus Aurelius

### The Dichotomy of Control

The most fundamental Stoic teaching is the **dichotomy of control**:

- **In our power**: judgements, impulses, desires, aversions
- **Not in our power**: body, reputation, command, wealth

### The Four Virtues

1. **Wisdom** — knowing what is good, bad, and indifferent
2. **Courage** — facing difficulties with equanimity
3. **Justice** — treating others fairly
4. **Temperance** — moderation in all things

## Key Figures

- [[Zeno of Citium]] — founder of Stoicism
- Epictetus — former slave, author of the *Enchiridion*
- Marcus Aurelius — Roman emperor, author of *Meditations*
- Seneca — Roman statesman and playwright

## Relationship to Other Schools

Stoicism shares some ground with [[Epistemology]] in its emphasis on reason and knowledge. It differs from Epicureanism in its view of pleasure and pain.

## Code Example

```python
def stoic_response(event: str) -> str:
    """Apply the dichotomy of control."""
    in_our_power = ["judgement", "response", "attitude"]
    if any(p in event for p in in_our_power):
        return "This is in our power — act virtuously."
    return "This is not in our power — accept it."
```

## See Also

- [[Epistemology]]
- [[Index]]
