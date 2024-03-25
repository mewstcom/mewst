# frozen_string_literal: true

class CreateHomeTimelinePosts < ActiveRecord::Migration[7.1]
  def change
    create_table :home_timeline_posts, id: false do |t|
      t.uuid :id, default: "generate_ulid()", null: false, primary_key: true
      t.references :profile, foreign_key: true, null: false, type: :uuid
      t.references :post, foreign_key: true, null: false, type: :uuid
      t.datetime :published_at, null: false

      t.index %i[profile_id post_id], unique: true
      t.index :published_at
    end
  end
end
