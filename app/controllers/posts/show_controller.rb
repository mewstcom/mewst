# typed: true
# frozen_string_literal: true

class Posts::ShowController < ApplicationController
  include ControllerConcerns::Authenticatable
  include ControllerConcerns::Localizable

  around_action :set_locale

  sig { returns(T.untyped) }
  def call
    @profile = ProfileRecord.kept.find_by!(atname: params[:atname])
    @post = @profile.post_records.kept.find(params[:post_id])
    @stamp_checker = StampChecker.new(profile: viewer&.profile_record, posts: [@post])
  end
end
