# frozen_string_literal: true

class AddGravatarUrlToProfiles < ActiveRecord::Migration[7.1]
  def change
    StrongMigrations.disable_check(:rename_column)
    rename_column :profiles, :avatar_url, :image_url

    safety_assured do
      change_table :profiles, bulk: true do |t|
        t.string :gravatar_email, default: "", null: false
        t.string :gravatar_url, default: "", null: false
        t.string :avatar_kind, default: "default", null: false
      end
    end
  end
end
