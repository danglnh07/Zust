# Requirement

## Zust 

- A video sharing social media platform 
- Feature: 

    1. Authentication 
    2. Account management
    3. Video management
    4. Feed management
    5. Video interaction
    6. Notification
    7. Search
    8. Statistics
    9. Report

## Actors

- Guest
- User
- Admin 

## Use cases

### Guest

- UC-01: Register
- UC-02: View video
- UC-03: View comments in a video's comment section
- UC-04: View feed
- UC-05: View profile
- UC-06: View list of uploaded video of an account
- UC-07: Search

### User

- Extends `Guest`
- UC-08: Login
- UC-09: Logout
- UC-10: Create video
- UC-11: Edit video
- UC-12: Delete video
- UC-13: Create a comment on a video's comment section
- UC-14: Edit a comment on a video's comment section
- UC-15: Delete a comment on a video's comment section
- UC-16: Like a comment on a video's comment section
- UC-17: Report a comment on a video's comment section
- UC-18: Like video
- UC-19: Share video
- UC-20: Add video to favourite list
- UC-21: Report video
- UC-22: Subscribe
- UC-23: Unsubscribe
- UC-24: Report account
- UC-25: Edit personal profile
- UC-26: Lock account
- UC-27: View watch history
- UC-28: View favorited video list
- UC-29: View liked video
- UC-30: View list of subscriber
- UC-31: View all notifications
- UC-32: Mark notification as read
- UC-33: Mark all notifications as read
- UC-34: View personal account's activity statistics
- UC-35: Download personal account's activity statistics

### Admin

- UC-36: View all video reports
- UC-37: View all comment reports
- UC-39: View all account reports
- UC-40: Process report
- UC-41: Force delete video
- UC-42: Force delete comment
- UC-43: Warn account
- UC-44: Ban account
- UC-45: View platform statistics ???

## Business rules

- BR-01: Except for feed (which use an algorithm), all list of videos will be sorted in descending order based on last update 
- BR-02: Statistic report can be download as csv format
- BR-03: A view is counted if the current user watch this video 40% of the total video time (exclude the time being fast-forward)
- BR-05: Any report get process by the admin must notify the owner of the reproted content through notification. 
In case the reported subject is account, there are 2 levels: warning (account get frozen in 10 days), 
and ban (the notification will be sent via registered email)
- BR-06: In edit video, you can only edit the video information (title, description, thumbnail). 
The video itself cannot be changed