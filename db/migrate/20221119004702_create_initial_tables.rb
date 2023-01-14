# frozen_string_literal: true

class CreateInitialTables < ActiveRecord::Migration[7.0]
  def change
    create_table :phone_number_verifications, id: false do |t|
      t.uuid :id, default: "generate_ulid()", null: false, primary_key: true
      t.string :phone_number, null: false
      t.string :raw_phone_number, null: false
      t.string :confirmation_code, null: false
      t.timestamps

      t.index %i[phone_number confirmation_code], name: :index_phone_number_verifications_on_pn_and_cc, unique: true
    end

    create_table :accounts, id: false do |t|
      t.uuid :id, default: "generate_ulid()", null: false, primary_key: true
      t.string :phone_number, index: {unique: true}, null: false
      t.string :locale, null: false
      t.integer :sign_in_count, default: 0, null: false
      t.timestamp :current_signed_in_at
      t.timestamp :last_signed_in_at
      t.timestamps
    end

    create_table :organizations, id: false do |t|
      t.uuid :id, default: "generate_ulid()", null: false, primary_key: true
      t.timestamps
    end

    create_table :organization_members, id: false do |t|
      t.uuid :id, default: "generate_ulid()", null: false, primary_key: true
      t.references :organization, foreign_key: true, null: false, type: :uuid
      t.references :account, foreign_key: true, null: false, type: :uuid
      t.timestamps

      t.index %i[organization_id account_id], unique: true
    end

    create_table :profiles, id: false do |t|
      t.uuid :id, default: "generate_ulid()", null: false, primary_key: true
      t.string :profilable_type, null: false
      t.citext :atname, index: {unique: true}, null: false
      t.string :name, default: "", null: false
      t.string :description, default: "", null: false
      t.text :avatar_data
      t.timestamp :deleted_at
      t.timestamps
    end

    create_table :account_profiles, id: false do |t|
      t.uuid :id, default: "generate_ulid()", null: false, primary_key: true
      t.references :account, foreign_key: true, index: {unique: true}, null: false, type: :uuid
      t.references :profile, foreign_key: true, index: {unique: true}, null: false, type: :uuid
      t.timestamps
    end

    create_table :organization_profiles, id: false do |t|
      t.uuid :id, default: "generate_ulid()", null: false, primary_key: true
      t.references :organization, foreign_key: true, null: false, type: :uuid
      t.references :profile, foreign_key: true, null: false, type: :uuid
      t.timestamps

      t.index %i[organization_id profile_id], unique: true
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

    create_table :twitter_accounts, id: false do |t|
      t.uuid :id, default: "generate_ulid()", null: false, primary_key: true
      t.references :profile, foreign_key: true, index: {unique: true}, null: false, type: :uuid
      t.string :uid, null: false
      t.string :username, null: false
      t.string :scopes, array: true, null: false
      t.string :access_token, null: false
      t.string :refresh_token, null: false
      t.timestamps
    end
  end
end
