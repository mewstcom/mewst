# frozen_string_literal: true

class AddRefreshTokenToTwitterAccounts < ActiveRecord::Migration[7.0]
  def change
    add_column :twitter_accounts, :refresh_token, :string, null: false
    add_column :twitter_accounts, :access_token_expired_at, :datetime, null: false
  end
end
