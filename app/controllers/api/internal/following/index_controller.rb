# typed: true
# frozen_string_literal: true

class Api::Internal::Following::IndexController < ApplicationController
  include Authenticatable

  before_action :require_authentication

  sig { returns(T.untyped) }
  def call
    idnames = params[:idnames]
    follows = current_user.follows.where(target_profile: Profile.only_kept.where(idname: idnames))
    following_idnames = follows.map(&:target_profile).map(&:idname)

    result = idnames.map do |idname|
      [idname, idname.in?(following_idnames)]
    end

    render(json: result.to_h)
  end
end
