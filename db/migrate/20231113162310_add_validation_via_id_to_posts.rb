# frozen_string_literal: true

class AddValidationViaIdToPosts < ActiveRecord::Migration[7.0]
  def change
    validate_foreign_key :posts, :oauth_applications
  end
end
