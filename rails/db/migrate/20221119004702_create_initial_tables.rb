# frozen_string_literal: true

class CreateInitialTables < ActiveRecord::Migration[7.0]
  def change
    create_table :email_confirmations, id: false do |t|
      t.uuid :id, default: "generate_ulid()", null: false, primary_key: true
      t.string :email, null: false
      t.string :event, null: false
      t.string :code, null: false
      t.timestamp :succeeded_at
      t.timestamps

      t.index :created_at
      t.index %i[email code], unique: true
    end

    create_table :profiles, id: false do |t|
      t.uuid :id, default: "generate_ulid()", null: false, primary_key: true
      t.string :profileable_type, null: false
      t.citext :atname, index: {unique: true}, null: false
      t.string :name, default: "", null: false
      t.string :description, default: "", null: false
      t.string :avatar_url, default: "", null: false
      t.timestamp :deleted_at
      t.timestamp :joined_at, null: false
      t.timestamps
    end

    create_table :users, id: false do |t|
      t.uuid :id, default: "generate_ulid()", null: false, primary_key: true
      t.references :profile, foreign_key: true, index: {unique: true}, null: false, type: :uuid
      t.string :email, index: {unique: true}, null: false
      t.string :password_digest, null: false
      t.string :locale, null: false
      t.integer :sign_in_count, default: 0, null: false
      t.timestamp :current_signed_in_at
      t.timestamp :last_signed_in_at
      t.timestamp :signed_up_at, null: false
      t.timestamps
    end

    create_table :actors, id: false do |t|
      t.uuid :id, default: "generate_ulid()", null: false, primary_key: true
      t.references :user, foreign_key: true, index: {unique: true}, null: false, type: :uuid
      t.references :profile, foreign_key: true, index: {unique: true}, null: false, type: :uuid
      t.timestamps

      t.index %i[user_id profile_id], unique: true
    end

    create_table :oauth_applications, id: false do |t|
      t.uuid :id, default: "generate_ulid()", null: false, primary_key: true
      t.string :name, index: {unique: true}, null: false
      t.string :uid, index: {unique: true}, null: false
      t.string :secret, null: false
      t.text :redirect_uri, null: false
      t.string :scopes, default: "", null: false
      t.boolean :confidential, default: true, null: false
      t.timestamps
    end

    create_table :oauth_access_grants, id: false do |t|
      t.uuid :id, default: "generate_ulid()", null: false, primary_key: true
      t.references :resource_owner, foreign_key: {to_table: :profiles}, null: false, type: :uuid
      t.references :user, foreign_key: true, null: false, type: :uuid
      t.references :application, foreign_key: {to_table: :oauth_applications}, null: false, type: :uuid
      t.string :token, index: {unique: true}, null: false
      t.integer :expires_in, null: false
      t.text :redirect_uri, null: false
      t.string :scopes, default: "", null: false
      t.datetime :created_at, null: false
      t.datetime :revoked_at
    end

    create_table :oauth_access_tokens, id: false do |t|
      t.uuid :id, default: "generate_ulid()", null: false, primary_key: true
      t.references :resource_owner, foreign_key: {to_table: :actors}, type: :uuid
      t.references :application, foreign_key: {to_table: :oauth_applications}, null: false, type: :uuid
      t.string :token, index: {unique: true}, null: false
      t.string :refresh_token, index: {unique: true}
      t.integer :expires_in
      t.string :scopes
      t.datetime :created_at, null: false
      t.datetime :revoked_at

      # The authorization server MAY issue a new refresh token, in which case
      # *the client MUST discard the old refresh token* and replace it with the
      # new refresh token. The authorization server MAY revoke the old
      # refresh token after issuing a new refresh token to the client.
      # @see https://datatracker.ietf.org/doc/html/rfc6749#section-6
      #
      # Doorkeeper implementation: if there is a `previous_refresh_token` column,
      # refresh tokens will be revoked after a related access token is used.
      # If there is no `previous_refresh_token` column, previous tokens are
      # revoked as soon as a new access token is created.
      #
      # Comment out this line if you want refresh tokens to be instantly
      # revoked after use.
      t.string :previous_refresh_token, default: "", null: false
    end

    create_table :follows, id: false do |t|
      t.uuid :id, default: "generate_ulid()", null: false, primary_key: true
      t.references :source_profile, foreign_key: {to_table: :profiles}, null: false, type: :uuid
      t.references :target_profile, foreign_key: {to_table: :profiles}, null: false, type: :uuid
      t.timestamp :followed_at, null: false
      t.timestamps

      t.index %i[source_profile_id target_profile_id], unique: true
    end

    create_table :posts, id: false do |t|
      t.uuid :id, default: "generate_ulid()", null: false, primary_key: true
      t.references :profile, foreign_key: true, null: false, type: :uuid
      t.text :comment, null: false
      t.integer :stamps_count, default: 0, null: false
      t.timestamp :published_at, null: false
      t.timestamps
    end

    create_table :stamps, id: false do |t|
      t.uuid :id, default: "generate_ulid()", null: false, primary_key: true
      t.references :profile, foreign_key: true, null: false, type: :uuid
      t.references :post, foreign_key: true, null: false, type: :uuid
      t.timestamp :stamped_at, null: false
      t.timestamps

      t.index %i[profile_id post_id], unique: true
    end
  end
end
