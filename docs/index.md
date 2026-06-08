---
title: Introduction
---
# 👋 Hi, Lets get started with triad

`triad` is a lightweight, idiomatic and composable router/wrapper for building Go HTTP services. It's especially good at helping you write large REST API services that are kept maintainable as your project grows and changes. `triad` is built on the top of Go 1.22 `http.ServeMux` to provide centralize error handling and Easy route registration.

The key considerations of triad's design are: project structure, maintainability, standard http handlers with error return, developer productivity, and deconstructing a large system into many small parts. The core router github.com/jshk00/triad is quite small (less than 1000 LOC), but we've also included some useful/optional utility functions for request and response. We hope you enjoy it too!

# Features

- Lightweight - cloc'd in ~1000 LOC for the chi router
- Fast - yes, see benchmarks
- 100% compatible with net/http - use any http or middleware pkg in the ecosystem that is also compatible with net/http
- Designed for modular/composable APIs - middlewares, inline middlewares, external and internal Groups for mounting
- No external dependencies - plain ol' Go stdlib + net/http
- Centralize error handling - error returning handlers no more need for naked return
- Optional utility functions such as Text, JSON, Stream, SSE
