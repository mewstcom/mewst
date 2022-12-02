# frozen_string_literal: true

class CreateInitialTables < ActiveRecord::Migration[7.0]
  def change
    create_table :users, id: :uuid do |t|
      t.string :phone_number, index: {unique: true}, null: false
      t.integer :sign_in_count, null: false, default: 0
      t.timestamp :current_signed_in_at
      t.timestamp :last_signed_in_at
      t.timestamps
    end

    create_table :email_confirmations, id: :uuid do |t|
      t.references :user, foreign_key: true, type: :uuid
      t.string :email, null: false
      t.integer :event, null: false
      t.string :token, index: {unique: true}, null: false
      t.string :back
      t.datetime :expires_at, null: false
      t.timestamps
    end

    create_table :organizations, id: :uuid do |t|
      t.timestamps
    end

    create_table :organization_members, id: :uuid do |t|
      t.references :organization, foreign_key: true, null: false, type: :uuid
      t.references :user, foreign_key: true, null: false, type: :uuid
      t.timestamps
    end
    add_index :organization_members, %i[organization_id user_id], unique: true

    create_table :profiles, id: :uuid do |t|
      t.integer :profilable_type, null: false
      t.citext :idname, null: false, index: {unique: true}
      t.string :name, null: false, default: ""
      t.string :description, null: false, default: ""
      t.timestamp :deleted_at
      t.timestamps
    end

    create_table :user_profiles, id: :uuid do |t|
      t.references :user, foreign_key: true, type: :uuid
      t.references :profile, foreign_key: true, type: :uuid
    end
    add_index :user_profiles, %i[user_id profile_id], unique: true

    create_table :organization_profiles, id: :uuid do |t|
      t.references :organization, foreign_key: true, type: :uuid
      t.references :profile, foreign_key: true, type: :uuid
    end
    add_index :organization_profiles, %i[organization_id profile_id], unique: true

    create_table :follows, id: :uuid do |t|
      t.references :source_profile, foreign_key: {to_table: :profiles}, null: false, type: :uuid
      t.references :target_profile, foreign_key: {to_table: :profiles}, null: false, type: :uuid
      t.timestamps
    end
    add_index :follows, %i[source_profile_id target_profile_id], unique: true

    create_table :posts, id: :uuid do |t|
      t.references :profile, foreign_key: true, null: false, type: :uuid
      t.text :body, null: false
      t.timestamps
    end
  end
end
