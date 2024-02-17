# typed: true
# frozen_string_literal: true

class Follows::CreateController < ApplicationController
  include ControllerConcerns::Authenticatable
  include ControllerConcerns::Localizable

  around_action :set_locale
  before_action :require_authentication

  sig { returns(T.untyped) }
  def call
    @form = FollowForm.new(profile: current_actor!.profile.not_nil!, target_atname: params[:atname])

    respond_to do |format|
      if @form.invalid?
        return format.turbo_stream { render(status: :unprocessable_entity) }
      end

      result = FollowProfileUseCase.new.call(profile: @form.profile.not_nil!, target_profile: @form.target_profile.not_nil!)

      @profile = result.target_profile
      @follow_checker = FollowChecker.new(profile: current_actor.profile.not_nil!, target_profiles: [@profile])

      format.turbo_stream
    end
  end
end
