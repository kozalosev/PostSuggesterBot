CREATE TABLE IF NOT EXISTS Suggestions(
    uid bigint NOT NULL REFERENCES Users(uid),
    message_id int NOT NULL,
    anonymously bool NOT NULL,
    published bool NOT NULL DEFAULT false,
    revoked bool NOT NULL DEFAULT false,

    PRIMARY KEY (uid, message_id)
);
