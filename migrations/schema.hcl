table "rate" {
  schema = schema.public
  column "id" {
    null = false
    type = text
  }
  column "price" {
    null = false
    type = text
  }
  column "first_signer" {
    null = false
    type = text
  }
  column "sign_data" {
    null = false
    type = text
  }
  column "lastsigned_time" {
    null = false
    type = timestamp
  }
  column "created_time" {//
//    null = false
    type = timestamp
  }
  primary_key {
    columns = [column.id]
  }
}
schema "public" {
  comment = "Default public rate schema"
}