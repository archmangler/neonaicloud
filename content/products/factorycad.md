---
title: AI Factory CAD Suite
slug: factorycad
status: published
summary: An enterprise CAD platform for AI Factory engineering—deterministic compilers that turn requirements into ranked, production-oriented architecture artefacts.
capabilities: ai-infrastructure, platform-engineering, cloud
updated: 2026-07-04
---

# Overview

The **AI Factory CAD Suite** (FactoryCAD) is Neon AI Cloud’s engineering platform for the design, analysis, and generation of **AI Factory architectures**.

Its core intellectual property is not a workflow engine or an LLM wrapper. It is a set of **deterministic engineering compilers** that encode AI Factory knowledge—compute, networking, reference architectures, rules, and validation—into repeatable design outputs.

## What it does

FactoryCAD transforms customer and programme requirements into structured engineering options:

- Accept and normalise architecture design requests
- Load **Compiler Knowledge Packages (CKPs)**—schemas, rules, templates, and acceptance cases
- Generate a **choice set** of feasible reference architectures (not a single opaque answer)
- Rank options with explicit, explainable criteria (scale fit, platform match, fabric support, operational fit, and related factors)
- Emit structured outputs for SME review, proposal support, and downstream engineering work

The first vertical slice is the **AI Factory Reference Architecture / Sizing Compiler**, focused on production-oriented AI Factory design paths such as InfiniBand-track reference architectures at meaningful GPU scale.

## Compiler-first philosophy

**Requirements → CKP rules and templates → Option generation → Deterministic ranking → Engineering artefacts**

Same input and same CKP produce the same ranking. There is no hidden randomness and no substitute of model improvisation for engineering rules.

## Who it is for

- AI infrastructure architects and systems integrators
- Consultancies designing AI Factories under real constraints
- Enterprises that need explainable architecture options, not a single black-box recommendation

## Engagement shape

Neon AI Cloud uses FactoryCAD in infrastructure design engagements to accelerate reference architecture selection, make trade-offs explicit, and leave clients with artefacts that can be validated and operated.

## Technical posture

Modular monolith, compiler-first architecture, CKP-driven development, library-before-API execution (CLI and UI invoke the same compiler). Built for engineering mastery: deterministic, reviewable, and extensible as new fabrics, platforms, and knowledge packages are added.
