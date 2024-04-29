# typed: true
# frozen_string_literal: true

class Follows::DestroyController < ApplicationController
  include ControllerConcerns::Authenticatable
  include ControllerConcerns::Localizable

  around_action :set_locale
  before_action :require_authentication

  sig { returns(T.untyped) }
  def call
    @form = FollowForm.new(profile: viewer!.profile.not_nil!, target_atname: params[:atname])

    if @form.invalid?
      return render("follows/create/call", content_type: "text/vnd.turbo-stream.html", layout: false, status: :unprocessable_entity)
    end

    result = UnfollowProfileUseCase.new.call(profile: @form.profile.not_nil!, target_profile: @form.target_profile.not_nil!)

    @profile = result.target_profile
    @follow_checker = FollowChecker.new(profile: viewer!.profile.not_nil!, target_profiles: [@profile])

    render("follows/create/call", content_type: "text/vnd.turbo-stream.html", layout: false)
  end
end
