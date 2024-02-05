# typed: true
# frozen_string_literal: true

class Search::ShowController < ApplicationController
  include ControllerConcerns::Authenticatable
  include ControllerConcerns::Localizable

  around_action :set_locale
  before_action :require_authentication

  sig { returns(T.untyped) }
  def call
    @profiles = current_actor!.suggested_followees.kept.merge(SuggestedFollow.not_checked).order(created_at: :desc).limit(30)
  end
end
