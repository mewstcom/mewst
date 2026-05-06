# frozen_string_literal: true

class AddNullFalseToOauthApplicationIdOnPosts < ActiveRecord::Migration[7.0]
  def change
    StrongMigrations.disable_check(:change_column_null_postgresql)
    change_column_null :posts, :oauth_application_id, false
  end
end
