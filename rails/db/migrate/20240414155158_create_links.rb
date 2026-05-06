# frozen_string_literal: true

class CreateLinks < ActiveRecord::Migration[7.1]
  def change
    create_table :links, id: false do |t|
      t.uuid :id, default: "generate_ulid()", null: false, primary_key: true
      t.string :canonical_url, index: {unique: true}, null: false
      t.string :domain, null: false
      t.string :title, null: false
      t.string :image_url, default: "", null: false
      t.timestamps
    end
  end
end
