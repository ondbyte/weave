content text "external_block" {
    text = "External block body"
}

document "some" {
    content ref _ {
        ref = content.text.external_block
        text = "More important text"
        other_field = "foobar"
    }
}

document "some2" {
    content "some3" _ {
        ref = content.text.external_block
        text = "More important text"
        other_field = "foobar"
    }
}