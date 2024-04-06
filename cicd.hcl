listen = ":4008"

repo "https://github.com/user1/project1" "main" "deploy" {
    secret = "secretme1"
    script = [ "/usr/local/bin/deploy1.sh" ]
}

repo "https://gitlab.com/user2/project2" "main" "deploy" {
    secret = "secretme2"
    script = [ "/usr/local/bin/deploy2.sh" ]
}
