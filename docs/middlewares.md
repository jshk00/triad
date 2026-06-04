---
title: Middleware
---
## Introduction
!!! Info
    Middleware performs some specific function on the HTTP request or response at a specific stage in the HTTP pipeline before or after the user defined controller.
    Middleware is a design pattern to eloquently add cross cutting concerns like logging, handling authentication without having many code contact points.

`triad's` middleware is `func(next  func(http.ResponseWriter, *Request) error) func(http.ResponseWriter, *Request) error` can be simplied to `func(next traid.HandlerFunc) traid.HanderFunc` or you can register stdlib net/http middleware handler of type `func(next http.Handler) http.Handler`. `Triad's` middleware are designed to be friendly and compatible with any middleware in community supporting net/http, There is nothing special about them except the fact the error handling done automatically when error returned from any point in middleware chain.

!!! Warn
    Middleware register with net/http compatiblity will not return error so handling error in middleware and writing it to response is responsibility of that middleware itself.
    To register the middleware using compatiblity mode you can use `(*Triad).Use(triad.CompatMiddleware(func(next http.Handler) http.handler))`. Furthermore all the middlewares needs to ne registered before adding any route else application my panic at build time.
