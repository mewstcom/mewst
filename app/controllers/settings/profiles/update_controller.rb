# typed: true
# frozen_string_literal: true

class Settings::Profiles::UpdateController < ApplicationController
  include Authenticatable
  include Localizable

  around_action :set_locale
  before_action :require_authentication

  sig { returns(T.untyped) }
  def call
    @profile = current_profile!
    @profile.attributes = profile_params

    if @profile.save
      flash[:notice] = t("messages.profiles.updated")
      redirect_to settings_path
    else
      render("settings/profiles/show/call", status: :unprocessable_entity)
    end
  end

  private

  sig { returns(ActionController::Parameters) }
  def profile_params
    T.cast(params.require(:profile), ActionController::Parameters).permit(:atname, :name, :description, :avatar)
  end
end
