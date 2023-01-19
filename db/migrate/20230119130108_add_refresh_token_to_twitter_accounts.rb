# frozen_string_literal: true

class AddRefreshTokenToTwitterAccounts < ActiveRecord::Migration[7.0]
  def change
    change_table :twitter_accounts, bulk: true do |t|
      t.string :refresh_token, null: false
      t.datetime :access_token_expired_at, null: false
    end
  end
end
