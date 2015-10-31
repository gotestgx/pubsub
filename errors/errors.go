package errors

const (
    Ok = iota
    AlreadySubscribed = iota
    NotSubscribed = iota
    NoNewMessage = iota
    NonExistentTopic = iota
)