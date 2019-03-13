# Hacker News Clone

This repository contains a very simple clone of the Hacker News (HN) website. It is my solution to the Gophercises [exercise #13](https://github.com/gophercises/quiet_hn). On this occasion, I chose to implement this exercise from scratch, as it didn't seem too difficult.

This HN implementation loads and caches 30 headlines using multiple goroutines to maximise loading speed. While the cache is valid, http requests to the web page will display the stored headlines; every 15 minutes the cached headlines are automatically refreshed.