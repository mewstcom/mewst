# frozen_string_literal: true

class CreateInitialTables < ActiveRecord::Migration[7.0]
  def change
    create_table :verifications, id: false do |t|
      t.uuid :id, default: "generate_ulid()", null: false, primary_key: true
      t.string :email, null: false
      t.string :event, null: false
      t.string :code, null: false
      t.timestamps

      t.index :created_at
      t.index %i[email code], unique: true
    end

    create_table :users, id: false do |t|
      t.uuid :id, default: "generate_ulid()", null: false, primary_key: true
      t.string :email, index: {unique: true}, null: false
      t.string :password_digest, null: false
      t.string :locale, null: false
      t.integer :sign_in_count, default: 0, null: false
      t.timestamp :current_signed_in_at
      t.timestamp :last_signed_in_at
      t.timestamps
    end

    create_table :profiles, id: false do |t|
      t.uuid :id, default: "generate_ulid()", null: false, primary_key: true
      t.citext :atname, index: {unique: true}, null: false
      t.string :name, default: "", null: false
      t.string :description, default: "", null: false
      t.string :avatar_url
      t.timestamp :deleted_at
      t.timestamps
    end

    create_table :profile_members, id: false do |t|
      t.uuid :id, default: "generate_ulid()", null: false, primary_key: true
      t.references :profile, foreign_key: true, null: false, type: :uuid
      t.references :user, foreign_key: true, null: false, type: :uuid
      t.timestamps

      t.index %i[profile_id user_id], unique: true
    end

    create_table :follows, id: false do |t|
      t.uuid :id, default: "generate_ulid()", null: false, primary_key: true
      t.references :source_profile, foreign_key: {to_table: :profiles}, null: false, type: :uuid
      t.references :target_profile, foreign_key: {to_table: :profiles}, null: false, type: :uuid
      t.timestamps

      t.index %i[source_profile_id target_profile_id], unique: true
    end

    create_table :posts, id: false do |t|
      t.uuid :id, default: "generate_ulid()", null: false, primary_key: true
      t.references :profile, foreign_key: true, null: false, type: :uuid
      t.text :content, null: false
      t.timestamps
    end
  end
end
