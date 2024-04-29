# typed: true
# frozen_string_literal: true

class Search::ShowController < ApplicationController
  include ControllerConcerns::Authenticatable
  include ControllerConcerns::Localizable

  around_action :set_locale
  before_action :require_authentication

  sig { returns(T.untyped) }
  def call
    @profiles = viewer!.checkable_suggested_followees.order(created_at: :desc).limit(30)
    @follow_checker = FollowChecker.new(profile: viewer!.profile.not_nil!, target_profiles: @profiles)
  end
end
