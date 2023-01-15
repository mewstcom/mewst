# typed: true
# frozen_string_literal: true

class Settings::TwitterAccounts::DestroyController < ApplicationController
  include Authenticatable
  include Localizable

  around_action :set_locale
  before_action :require_authentication

  sig { returns(T.untyped) }
  def call
    current_profile!.twitter_account&.destroy!

    flash[:success] = t("messages.twitter_accounts.disconnected")
    redirect_back(fallback_location: settings_connected_account_list_path)
  end
end
