# frozen_string_literal: true

class RemoveSignInColumnsOnUsers < ActiveRecord::Migration[7.1]
  def change
    safety_assured do
      remove_column :users, :sign_in_count
      remove_column :users, :current_signed_in_at
      remove_column :users, :last_signed_in_at
    end
  end
end
