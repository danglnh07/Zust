-- name: CreateVideo :one
INSERT INTO video (title, duration, description, publisher_id)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: GetVideo :one
SELECT 
    v.video_id, v.title, v.duration, v.description, v.created_at,
    a.account_id, a.username,
    (SELECT COUNT(*) FROM subscribe s WHERE s.subscribe_to_id = v.publisher_id) AS total_subscriber,
    (SELECT COUNT(*) FROM watch_video wv WHERE wv.video_id = v.video_id) AS total_view,
    (SELECT COUNT(*) FROM like_video lv WHERE lv.video_id = v.video_id) AS total_like
FROM video v 
JOIN account a ON a.account_id = v.publisher_id
WHERE v.video_id = $1;
