# typed: true
# frozen_string_literal: true

class Api::Internal::Following::IndexController < ApplicationController
  include Authenticatable

  before_action :require_authentication

  sig { returns(T.untyped) }
  def call
    if params[:idnames].nil?
      render(json: {}, status: :not_found)
    end

    idnames = T.cast(params[:idnames], T::Array[String])
    follows = T.must(current_profile).follows.where(target_profile: Profile.only_kept.where(idname: idnames))
    following_idnames = follows.map(&:target_profile).map { T.must(_1).idname }

    result = idnames.map do |idname|
      [idname, idname.in?(following_idnames)]
    end

    render(json: result.to_h)
  end
end
