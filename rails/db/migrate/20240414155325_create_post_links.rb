# frozen_string_literal: true

class CreatePostLinks < ActiveRecord::Migration[7.1]
  def change
    create_table :post_links, id: false do |t|
      t.uuid :id, default: "generate_ulid()", null: false, primary_key: true
      t.references :post, foreign_key: true, null: false, type: :uuid
      t.references :link, foreign_key: true, null: false, type: :uuid
      t.timestamps
    end
  end
end
