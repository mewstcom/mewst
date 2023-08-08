# typed: strict
# frozen_string_literal: true

class Latest::ResponseErrorSerializer < Latest::ApplicationSerializer
  attributes :code, :field, :message
end
