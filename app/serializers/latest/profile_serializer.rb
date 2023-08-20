# typed: false
# frozen_string_literal: true

class Latest::ProfileSerializer < Latest::ApplicationSerializer
  root_key :profile, :profiles

  attributes :atname, :avatar_url, :name, :viewer_has_followed
end
