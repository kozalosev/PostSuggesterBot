CREATE TABLE IF NOT EXISTS Approvals(
    author_uid bigint,
    message_id int,

    approved_by bigint NOT NULL REFERENCES Users(uid),

    PRIMARY KEY (author_uid, message_id, approved_by),
    FOREIGN KEY (author_uid, message_id) REFERENCES Suggestions(uid, message_id)
);

CREATE INDEX IF NOT EXISTS idx_approvals ON Approvals(author_uid, message_id);
